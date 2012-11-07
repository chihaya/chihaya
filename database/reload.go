package database

import (
	"chihaya/config"
	"log"
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

		start := time.Now()

		var err error
		var count uint
		rows := db.query(db.loadUsersStmt)

		newUsers := make(map[string]*User)

		db.UsersMutex.Lock()
		for rows.Next() {
			u := &User{}
			var torrentPass string
			err = rows.Scan(&u.Id, &torrentPass, &u.DownMultiplier, &u.UpMultiplier, &u.Slots)
			if err != nil {
				log.Panicf("Error scanning user rows: %v", err)
			}

			old, exists := db.Users[torrentPass]
			if exists && old != nil {
				old.Id = u.Id
				old.DownMultiplier = u.DownMultiplier
				old.UpMultiplier = u.UpMultiplier
				old.Slots = u.Slots
				newUsers[torrentPass] = old
			} else {
				newUsers[torrentPass] = u
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

		start := time.Now()

		var err error
		var count uint
		rows := db.query(db.loadTorrentsStmt)

		newTorrents := make(map[string]*Torrent)

		db.TorrentsMutex.Lock()

		for rows.Next() {
			t := &Torrent{}
			var infoHash string
			err = rows.Scan(&t.Id, &infoHash, &t.DownMultiplier, &t.UpMultiplier, &t.Snatched)
			if err != nil {
				log.Panicf("Error scanning torrent rows: %v", err)
			}

			old, exists := db.Torrents[infoHash]
			if exists && old != nil {
				old.Id = t.Id
				old.DownMultiplier = t.DownMultiplier
				old.UpMultiplier = t.UpMultiplier
				old.Snatched = t.Snatched
				newTorrents[infoHash] = old
			} else {
				t.Seeders = make(map[string]*Peer)
				t.Leechers = make(map[string]*Peer)
				newTorrents[infoHash] = t
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

		start := time.Now()

		var err error
		var count int
		rows := db.query(db.loadWhitelistStmt)

		db.WhitelistMutex.Lock()
		db.Whitelist = db.Whitelist[0:1] // Effectively clear the whitelist

		for rows.Next() {
			var peerId string
			err = rows.Scan(&peerId)
			if err != nil {
				log.Panicf("Error scanning whitelist rows: %v", err)
			}
			if count >= cap(db.Whitelist) {
				newSlice := make([]string, count, count*2)
				copy(newSlice, db.Whitelist)
				db.Whitelist = newSlice
			} else if count >= len(db.Whitelist) {
				db.Whitelist = db.Whitelist[0 : count+1]
			}
			db.Whitelist[count] = peerId
			count++
		}

		db.WhitelistMutex.Unlock()
		db.waitGroup.Done()

		log.Printf("Whitelist load complete (%d rows, %dms)", count, time.Now().Sub(start).Nanoseconds()/1000000)
		time.Sleep(config.DatabaseReloadInterval * 10)
	}
}
