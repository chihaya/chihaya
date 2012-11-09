package database

import (
	"chihaya/config"
	"log"
	"io"
	"time"
)

/*
 * Reloading is done asynchronously for each object cache.
 * Cache synchronization is handled by using sync.RWMutex, which has a bunch of advantages:
 *   - The number of simultaneous readers is arbitrarily high
 *   - Writing is blocked until all current readers release the mutex
 *   - Once a writer locks the mutex, new readers block until the writer unlocks it
 *
 * When writing, mutexes are only locked after the database call returns, so contention should be minimal.
 */
func (db *Database) startReloading() {
	go db.loadUsers()
	go db.loadTorrents()
	go db.loadWhitelist()
}

func (db *Database) loadUsers() {
	for !db.terminate {
		db.waitGroup.Add(1)

		var err error
		var count uint
		result := db.query(db.loadUsersStmt)
		start := time.Now()

		newUsers := make(map[string]*User, len(db.Users))

		row := &rowWrapper{result.MakeRow()}

		id := result.Map("ID")
		torrentPass := result.Map("torrent_pass")
		downMultiplier := result.Map("DownMultiplier")
		upMultiplier := result.Map("UpMultiplier")
		slots := result.Map("Slots")

		db.UsersMutex.Lock()

		for {
			err = result.ScanRow(row.r)
			if err == io.EOF {
				break
			} else if err != nil {
				log.Panicf("Error scanning user rows: %v", err)
			}

			torrentPass := row.Str(torrentPass)

			old, exists := db.Users[torrentPass]
			if exists && old != nil {
				old.Id = row.Uint64(id)
				old.DownMultiplier = row.Float64(downMultiplier)
				old.UpMultiplier = row.Float64(upMultiplier)
				old.Slots = row.Int64(slots)
				newUsers[torrentPass] = old
			} else {
				newUsers[torrentPass] = &User{
					Id: row.Uint64(id),
					UpMultiplier: row.Float64(downMultiplier),
					DownMultiplier: row.Float64(upMultiplier),
					Slots: row.Int64(slots),
					UsedSlots: 0,
				}
			}
			count++
		}

		db.Users = newUsers
		db.UsersMutex.Unlock()
		db.waitGroup.Done()

		log.Printf("User load complete (%d rows, %dms)", count, time.Now().Sub(start).Nanoseconds()/1000000)
		time.Sleep(config.DatabaseReloadInterval)
	}
}

func (db *Database) loadTorrents() {
	for !db.terminate {
		db.waitGroup.Add(1)

		var err error
		var count uint
		result := db.query(db.loadTorrentsStmt)
		start := time.Now()

		newTorrents := make(map[string]*Torrent)

		row := &rowWrapper{result.MakeRow()}

		id := result.Map("ID")
		infoHash := result.Map("info_hash")
		downMultiplier := result.Map("DownMultiplier")
		upMultiplier := result.Map("UpMultiplier")
		snatched := result.Map("Snatched")

		db.TorrentsMutex.Lock()

		for {
			err = result.ScanRow(row.r)
			if err == io.EOF {
				break
			} else if err != nil {
				log.Panicf("Error scanning torrent rows: %v", err)
			}

			infoHash := row.Str(infoHash)

			old, exists := db.Torrents[infoHash]
			if exists && old != nil {
				old.Id = row.Uint64(id)
				old.DownMultiplier = row.Float64(downMultiplier)
				old.UpMultiplier = row.Float64(upMultiplier)
				old.Snatched = row.Uint(snatched)
				newTorrents[infoHash] = old
			} else {
				newTorrents[infoHash] = &Torrent{
					Id: row.Uint64(id),
					UpMultiplier: row.Float64(downMultiplier),
					DownMultiplier: row.Float64(upMultiplier),
					Snatched: row.Uint(snatched),

					Seeders: make(map[string]*Peer),
					Leechers: make(map[string]*Peer),
				}
			}
			count++
		}

		db.Torrents = newTorrents
		db.TorrentsMutex.Unlock()
		db.waitGroup.Done()

		log.Printf("Torrent load complete (%d rows, %dms)", count, time.Now().Sub(start).Nanoseconds()/1000000)
		time.Sleep(config.DatabaseReloadInterval)
	}
}

func (db *Database) loadWhitelist() {
	for !db.terminate {
		db.waitGroup.Add(1)

		var err error
		var count int
		result := db.query(db.loadWhitelistStmt)
		start := time.Now()

		row := result.MakeRow()

		db.WhitelistMutex.Lock()
		db.Whitelist = db.Whitelist[0:1] // Effectively clear the whitelist

		for {
			err = result.ScanRow(row)
			if err == io.EOF {
				break
			} else if err != nil {
				log.Panicf("Error scanning whitelist rows: %v", err)
			}
			if count >= cap(db.Whitelist) {
				newSlice := make([]string, count, count*2)
				copy(newSlice, db.Whitelist)
				db.Whitelist = newSlice
			} else if count >= len(db.Whitelist) {
				db.Whitelist = db.Whitelist[0 : count+1]
			}
			db.Whitelist[count] = row.Str(0)
			count++
		}

		db.WhitelistMutex.Unlock()
		db.waitGroup.Done()

		log.Printf("Whitelist load complete (%d rows, %dms)", count, time.Now().Sub(start).Nanoseconds()/1000000)
		time.Sleep(config.DatabaseReloadInterval * 10)
	}
}
