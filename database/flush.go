// This file is part of Chihaya.
//
// Chihaya is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Chihaya is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Chihaya.  If not, see <http://www.gnu.org/licenses/>.

package database

import (
	"bytes"
	"log"
	"sync/atomic"
	"time"

	"github.com/kotokoko/chihaya/config"
	"github.com/kotokoko/chihaya/util"
)

/*
 * Channels are used for flushing to limit throughput to a manageable level.
 * If a client causes an update that requires a flush, it writes to the channel requesting that a flush occur.
 * However, if the channel is already full (to xFlushBufferSize), the client thread blocks until a flush occurs.
 * This way, rather than thrashing and missing flushes, clients are simply forced to wait longer.
 *
 * This tradeoff can be adjusted by tweaking the various xFlushBufferSize values to suit the server.
 *
 * Each flush routine now gets its own database connection to maximize update throughput.
 */

/*
 * If a buffer channel is less than half full on a flush, the routine will wait some time before flushing again.
 * If the channel is more than half full, it doesn't wait at all.
 * TODO: investigate good wait timings
 */

func (db *Database) startFlushing() {
	db.torrentChannel = make(chan *bytes.Buffer, config.Config.FlushSizes.Torrent)
	db.userChannel = make(chan *bytes.Buffer, config.Config.FlushSizes.User)
	db.transferHistoryChannel = make(chan *bytes.Buffer, config.Config.FlushSizes.TransferHistory)
	db.transferIpsChannel = make(chan *bytes.Buffer, config.Config.FlushSizes.TransferIps)
	db.snatchChannel = make(chan *bytes.Buffer, config.Config.FlushSizes.Snatch)
	db.slotVerificationChannel = make(chan *User, 100)

	go db.flushTorrents()
	go db.flushUsers()
	go db.flushTransferHistory()
	go db.flushTransferIps()
	go db.flushSnatches()

	go db.purgeInactivePeers()
	go db.startUsedSlotsVerification()
}

func (db *Database) flushTorrents() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int
	conn := OpenDatabaseConnection()

	for {
		length := util.MaxInt(1, len(db.torrentChannel))
		query.Reset()

		query.WriteString("INSERT INTO torrents (ID, Snatched, Seeders, Leechers, last_action) VALUES\n")

		for count = 0; count < length; count++ {
			b := <-db.torrentChannel
			if b == nil {
				break
			}
			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		if config.Config.LogFlushes && !db.terminate {
			log.Printf("[torrents] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE Snatched = Snatched + VALUES(Snatched), " +
				"Seeders = VALUES(Seeders), Leechers = VALUES(Leechers), " +
				"last_action = IF(last_action < VALUES(last_action), VALUES(last_action), last_action);")

			conn.execBuffer(&query)

			if length < (config.Config.FlushSizes.Torrent >> 1) {
				time.Sleep(config.Config.Intervals.FlushSleep.Duration)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	conn.Close()
}

func (db *Database) flushUsers() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int
	conn := OpenDatabaseConnection()

	for {
		length := util.MaxInt(1, len(db.userChannel))
		query.Reset()

		query.WriteString("INSERT INTO users_main (ID, Uploaded, Downloaded, rawdl, rawup) VALUES\n")

		for count = 0; count < length; count++ {
			b := <-db.userChannel
			if b == nil {
				break
			}
			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		if config.Config.LogFlushes && !db.terminate {
			log.Printf("[users_main] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE Uploaded = Uploaded + VALUES(Uploaded), " +
				"Downloaded = Downloaded + VALUES(Downloaded), rawdl = rawdl + VALUES(rawdl), rawup = rawup + VALUES(rawup);")

			conn.execBuffer(&query)

			if length < (config.Config.FlushSizes.User >> 1) {
				time.Sleep(config.Config.Intervals.FlushSleep.Duration)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	conn.Close()
}

func (db *Database) flushTransferHistory() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int
	conn := OpenDatabaseConnection()

	for {
		db.transferHistoryWaitGroup.Add(1)
		length := util.MaxInt(1, len(db.transferHistoryChannel))
		query.Reset()

		query.WriteString("INSERT INTO transfer_history (uid, fid, uploaded, downloaded, " +
			"seeding, starttime, last_announce, seedtime, active, snatched, remaining) VALUES\n")

		for count = 0; count < length; count++ {
			b := <-db.transferHistoryChannel
			if b == nil {
				break
			}
			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		if config.Config.LogFlushes && !db.terminate {
			log.Printf("[transfer_history] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE uploaded = uploaded + VALUES(uploaded), " +
				"downloaded = downloaded + VALUES(downloaded), connectable = VALUES(connectable), " +
				"seeding = VALUES(seeding), seedtime = seedtime + VALUES(seedtime), last_announce = VALUES(last_announce), " +
				"active = VALUES(active), snatched = snatched + VALUES(snatched), remaining = VALUES(remaining);")

			conn.execBuffer(&query)
			db.transferHistoryWaitGroup.Done()

			if length < (config.Config.FlushSizes.TransferHistory >> 1) {
				time.Sleep(config.Config.Intervals.FlushSleep.Duration)
			}
		} else if db.terminate {
			db.transferHistoryWaitGroup.Done()
			break
		} else {
			db.transferHistoryWaitGroup.Done()
			time.Sleep(time.Second)
		}
	}

	conn.Close()
}

func (db *Database) flushTransferIps() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int
	conn := OpenDatabaseConnection()

	for {
		length := util.MaxInt(1, len(db.transferIpsChannel))
		query.Reset()

		query.WriteString("INSERT INTO transfer_ips (uid, fid, peer_id, starttime, ip, port) VALUES\n")

		for count = 0; count < length; count++ {
			b := <-db.transferIpsChannel
			if b == nil {
				break
			}
			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		if config.Config.LogFlushes && !db.terminate {
			log.Printf("[transfer_ips] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE ip = VALUES(ip), port = VALUES(port);")

			conn.execBuffer(&query)

			if length < (config.Config.FlushSizes.TransferIps >> 1) {
				time.Sleep(config.Config.Intervals.FlushSleep.Duration)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	conn.Close()
}

func (db *Database) flushSnatches() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int
	conn := OpenDatabaseConnection()

	for {
		length := util.MaxInt(1, len(db.snatchChannel))
		query.Reset()

		query.WriteString("INSERT INTO transfer_history (uid, fid, snatched_time) VALUES\n")

		for count = 0; count < length; count++ {
			b := <-db.snatchChannel
			if b == nil {
				break
			}
			query.Write(b.Bytes())
			db.bufferPool.Give(b)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		if config.Config.LogFlushes && !db.terminate {
			log.Printf("[snatches] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE snatched_time = VALUES(snatched_time);")

			conn.execBuffer(&query)

			if length < (config.Config.FlushSizes.Snatch >> 1) {
				time.Sleep(config.Config.Intervals.FlushSleep.Duration)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	conn.Close()
}

func (db *Database) purgeInactivePeers() {
	time.Sleep(2 * time.Second)

	for !db.terminate {
		db.waitGroup.Add(1)

		start := time.Now()
		now := start.Unix()
		count := 0

		oldestActive := now - 2*int64(config.Config.Intervals.Announce.Seconds())

		// First, remove inactive peers from memory
		db.TorrentsMutex.Lock()
		for _, torrent := range db.Torrents {
			countThisTorrent := count
			for id, peer := range torrent.Leechers {
				if peer.LastAnnounce < oldestActive {
					delete(torrent.Leechers, id)

					// TODO: possibly optimize this
					for _, user := range db.Users {
						if user.Id == peer.UserId {
							atomic.AddInt64(&user.UsedSlots, -1)
							break
						}
					}
					count++
				}
			}
			for id, peer := range torrent.Seeders {
				if peer.LastAnnounce < oldestActive {
					delete(torrent.Seeders, id)
					count++
				}
			}
			if countThisTorrent != count {
				db.RecordTorrent(torrent, 0)
			}
		}
		db.TorrentsMutex.Unlock()

		log.Printf("Purged %d inactive peers from memory (%dms)\n", count, time.Now().Sub(start).Nanoseconds()/1000000)

		// Wait on flushing to prevent a race condition where the user has announced but their announce time hasn't been flushed yet
		db.transferHistoryWaitGroup.Wait()

		// Then set them to inactive in the database
		db.mainConn.mutex.Lock()
		start = time.Now()
		result := db.mainConn.exec(db.cleanStalePeersStmt, oldestActive)
		rows := result.AffectedRows()
		db.mainConn.mutex.Unlock()

		if rows > 0 {
			log.Printf("Updated %d inactive peers in database (%dms)\n", rows, time.Now().Sub(start).Nanoseconds()/1000000)
		}

		db.waitGroup.Done()
		time.Sleep(config.Config.Intervals.PurgeInactive.Duration)
	}
}

/*
 * Deleting a torrent that a user is leeching will cause their slot count to be incorrect,
 * so the count is verified every so often
 */
func (db *Database) startUsedSlotsVerification() {
	if !config.Config.SlotsEnabled {
		log.Printf("Slots disabled, skipping slot verification")
		return
	}

	var slots int64
	for !db.terminate {
		user := <-db.slotVerificationChannel
		if user == nil {
			break
		}
		if user.Slots == -1 {
			continue
		}

		db.waitGroup.Add(1)
		userId := user.Id

		slots = 0
		db.TorrentsMutex.RLock()
		for _, torrent := range db.Torrents {
			for _, peer := range torrent.Leechers {
				if peer.UserId == userId {
					slots++
				}
			}
		}
		db.TorrentsMutex.RUnlock()
		if user.UsedSlots != slots {
			if user.UsedSlots < slots {
				log.Printf("!!! WARNING/BUG !!! Negative UsedSlots value (%d < %d) for user %d\n", user.UsedSlots, slots, user.Id)
			}
			atomic.StoreInt64(&user.UsedSlots, slots)
			log.Printf("Fixed used slot cache for user %d", userId)
		}

		db.waitGroup.Done()
	}
}
