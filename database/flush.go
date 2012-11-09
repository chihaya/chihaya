package database

import (
	"bytes"
	"chihaya/config"
	"chihaya/util"
	"log"
	"runtime"
	"sync/atomic"
	"time"
)

/*
 * Channels are used for flushing to limit throughput to a manageable level.
 * If a client causes an update that requires a flush, it writes to the channel requesting that a flush occur.
 * However, if the channel is already full (to xFlushBufferSize), the client thread blocks until a flush occurs.
 * This way, rather than thrashing and missing flushes, clients are simply forced to wait longer.
 *
 * This tradeoff can be adjusted by tweaking the various xFlushBufferSize values to suit the server.
 */

/*
 * If a buffer channel is less than half full on a flush, the routine will wait some time before flushing again.
 * If the channel is more than half full, it doesn't wait at all.
 * TODO: investigate good wait timings
 */

func (db *Database) startFlushing() {
	db.torrentChannel = make(chan *bytes.Buffer, config.TorrentFlushBufferSize)
	db.userChannel = make(chan *bytes.Buffer, config.UserFlushBufferSize)
	db.transferHistoryChannel = make(chan *bytes.Buffer, config.TransferHistoryFlushBufferSize)
	db.transferIpsChannel = make(chan *bytes.Buffer, config.TransferIpsFlushBufferSize)
	db.snatchChannel = make(chan *bytes.Buffer, config.SnatchFlushBufferSize)

	go db.flushTorrents()
	go db.flushUsers()
	go db.flushTransferHistory()
	go db.flushTransferIps()
	go db.flushSnatches()

	go db.purgeInactivePeers()
	go db.verifyUsedSlotsCache()
}

func (db *Database) flushTorrents() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int

	for {
		length := util.Max(1, len(db.torrentChannel))
		query.Reset()

		query.WriteString("INSERT INTO torrents (ID, Snatched, Seeders, Leechers) VALUES\n")

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

		if config.LogFlushes && !db.terminate {
			log.Printf("[torrents] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE Snatched = Snatched + VALUES(Snatched), " +
				"Seeders = VALUES(Seeders), Leechers = VALUES(Leechers);")

			db.execBuffer(&query)

			if length < (config.TorrentFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (db *Database) flushUsers() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int

	for {
		length := util.Max(1, len(db.userChannel))
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

		if config.LogFlushes && !db.terminate {
			log.Printf("[users_main] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE Uploaded = Uploaded + VALUES(Uploaded), " +
				"Downloaded = Downloaded + VALUES(Downloaded), rawdl = rawdl + VALUES(rawdl), rawup = rawup + VALUES(rawup);")

			db.execBuffer(&query)

			if length < (config.UserFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (db *Database) flushTransferHistory() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int

	for {
		length := util.Max(1, len(db.transferHistoryChannel))
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

		if config.LogFlushes && !db.terminate {
			log.Printf("[transfer_history] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE uploaded = uploaded + VALUES(uploaded), " +
				"downloaded = downloaded + VALUES(downloaded), connectable = VALUES(connectable), " +
				"seeding = VALUES(seeding), seedtime = seedtime + VALUES(seedtime), last_announce = VALUES(last_announce), " +
				"active = VALUES(active), snatched = snatched + VALUES(snatched), remaining = VALUES(remaining);")

			db.execBuffer(&query)

			if length < (config.TransferHistoryFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (db *Database) flushTransferIps() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int

	for {
		length := util.Max(1, len(db.transferIpsChannel))
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

		if config.LogFlushes && !db.terminate {
			log.Printf("[transfer_ips] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE ip = VALUES(ip), port = VALUES(port);")

			db.execBuffer(&query)

			if length < (config.TransferIpsFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (db *Database) flushSnatches() {
	var query bytes.Buffer
	db.waitGroup.Add(1)
	defer db.waitGroup.Done()
	var count int

	for {
		length := util.Max(1, len(db.snatchChannel))
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

		if config.LogFlushes && !db.terminate {
			log.Printf("[snatches] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE snatched_time = VALUES(snatched_time);")

			db.execBuffer(&query)

			if length < (config.SnatchFlushBufferSize >> 1) {
				time.Sleep(config.FlushSleepInterval)
			}
		} else if db.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (db *Database) purgeInactivePeers() {
	for !db.terminate {
		db.waitGroup.Add(1)

		start := time.Now()
		now := start.Unix()
		count := 0

		oldestActive := now - 2*int64(config.AnnounceInterval.Seconds())

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

		log.Printf("Purged %d inactive peers from memory (%d ms)\n", count, time.Now().Sub(start).Nanoseconds()/1000000)

		// Then set them to inactive in the database
		result := db.exec(db.cleanStalePeersStmt, oldestActive)
		rows := result.AffectedRows()

		if rows > 0 {
			log.Printf("Updated %d inactive peers in database\n", rows)
		}

		db.waitGroup.Done()
		time.Sleep(config.PurgeInactiveInterval)
	}
}

/*
 * This is here because deleting a torrent that a user is leeching will cause their slot count to be incorrect
 *
 * Note that when there are *a lot* of leechers, this can get SLOW. TODO: make this function unnecessary
 */
func (db *Database) verifyUsedSlotsCache() {
	if !config.SlotsEnabled {
		log.Printf("Slots disabled, skipping slot verification")
		return
	}
	var slots int64
	for !db.terminate {
		db.waitGroup.Add(1)
		start := time.Now()

		log.Printf("Started verifying used slot cache (this may take a while)")

		inconsistent := 0

		db.UsersMutex.RLock()
		for _, user := range db.Users {
			if user.Slots == -1 {
				// Although slots used are still calculated for users with no restriction,
				// we don't care as much about consistency for them. If they suddenly get a restriction,
				// their slot count will be cleaned up on the next verification
				continue
			}
			slots = 0
			db.TorrentsMutex.RLock() // Unlock and lock in here to allow pending requests to continue processing
			for _, torrent := range db.Torrents {
				for _, peer := range torrent.Leechers {
					if peer.UserId == user.Id {
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
				inconsistent++
			}
			runtime.Gosched() // Yield because this is low priority
		}
		db.UsersMutex.RUnlock()

		log.Printf("Slot cache verification complete (%d fixed, %dms)\n", inconsistent, time.Now().Sub(start).Nanoseconds()/1000000)
		db.waitGroup.Done()
		time.Sleep(config.VerifyUsedSlotsInterval)
	}
}
