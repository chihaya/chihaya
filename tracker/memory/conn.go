// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"runtime"
	"time"

	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker/models"
)

// Conn implements a connection to a memory-based tracker data store.
type Conn struct {
	*Pool
}

func (c *Conn) Close() error {
	return nil
}

func (c *Conn) FindUser(passkey string) (*models.User, error) {
	c.usersM.RLock()
	defer c.usersM.RUnlock()

	user, exists := c.users[passkey]
	if !exists {
		return nil, models.ErrUserDNE
	}
	return &*user, nil
}

func (c *Conn) FindTorrent(infohash string) (*models.Torrent, error) {
	c.torrentsM.RLock()
	defer c.torrentsM.RUnlock()

	torrent, exists := c.torrents[infohash]
	if !exists {
		return nil, models.ErrTorrentDNE
	}
	return &*torrent, nil
}

func (c *Conn) FindClient(peerID string) error {
	c.whitelistM.RLock()
	defer c.whitelistM.RUnlock()

	_, ok := c.whitelist[peerID]
	if !ok {
		return models.ErrClientUnapproved
	}
	return nil
}

func (c *Conn) IncrementTorrentSnatches(infohash string) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, exists := c.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}
	t.Snatches++

	return nil
}

func (c *Conn) IncrementUserSnatches(userID string) error {
	c.usersM.Lock()
	defer c.usersM.Unlock()

	u, exists := c.users[userID]
	if !exists {
		return models.ErrUserDNE
	}
	u.Snatches++

	return nil
}

func (c *Conn) TouchTorrent(infohash string) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return models.ErrTorrentDNE
	}
	t.LastAction = time.Now().Unix()

	return nil
}

func (c *Conn) AddLeecher(infohash string, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return models.ErrTorrentDNE
	}
	t.Leechers[p.ID] = *p

	return nil
}

func (c *Conn) AddSeeder(infohash string, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return models.ErrTorrentDNE
	}
	t.Seeders[p.ID] = *p

	return nil
}

func (c *Conn) DeleteLeecher(infohash, peerkey string) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return models.ErrTorrentDNE
	}
	delete(t.Leechers, peerkey)

	return nil
}

func (c *Conn) DeleteSeeder(infohash, peerkey string) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return models.ErrTorrentDNE
	}
	delete(t.Seeders, peerkey)

	return nil
}

func (c *Conn) PutLeecher(infohash string, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return models.ErrTorrentDNE
	}
	t.Leechers[p.ID] = *p

	return nil
}

func (c *Conn) PutSeeder(infohash string, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return models.ErrTorrentDNE
	}
	t.Seeders[p.ID] = *p

	return nil
}

func (c *Conn) PutTorrent(t *models.Torrent) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	c.torrents[t.Infohash] = &*t

	return nil
}

func (c *Conn) DeleteTorrent(infohash string) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	delete(c.torrents, infohash)

	return nil
}

func (c *Conn) PurgeInactiveTorrent(infohash string) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent, exists := c.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	if torrent.PeerCount() == 0 {
		delete(c.torrents, infohash)
	}

	return nil
}

func (c *Conn) PutUser(u *models.User) error {
	c.usersM.Lock()
	defer c.usersM.Unlock()

	c.users[u.Passkey] = &*u

	return nil
}

func (c *Conn) DeleteUser(passkey string) error {
	c.usersM.Lock()
	defer c.usersM.Unlock()

	delete(c.users, passkey)

	return nil
}

func (c *Conn) PutClient(peerID string) error {
	c.whitelistM.Lock()
	defer c.whitelistM.Unlock()

	c.whitelist[peerID] = true

	return nil
}

func (c *Conn) DeleteClient(peerID string) error {
	c.whitelistM.Lock()
	defer c.whitelistM.Unlock()

	delete(c.whitelist, peerID)

	return nil
}

func (c *Conn) PurgeInactivePeers(purgeEmptyTorrents bool, before time.Time) error {
	unixtime := before.Unix()

	// Build array of map keys to operate on.
	c.torrentsM.RLock()
	index := 0
	keys := make([]string, len(c.torrents))

	for infohash, _ := range c.torrents {
		keys[index] = infohash
		index++
	}

	c.torrentsM.RUnlock()

	// Process keys.
	for _, infohash := range keys {
		runtime.Gosched() // Let other goroutines run, since this is low priority.

		c.torrentsM.Lock()
		torrent := c.torrents[infohash]

		if torrent == nil {
			continue // Torrent deleted since keys were computed.
		}

		for key, peer := range torrent.Seeders {
			if peer.LastAnnounce <= unixtime {
				delete(torrent.Seeders, key)
				stats.RecordPeerEvent(stats.ReapedSeed, peer.HasIPv6())
			}
		}

		for key, peer := range torrent.Leechers {
			if peer.LastAnnounce <= unixtime {
				delete(torrent.Leechers, key)
				stats.RecordPeerEvent(stats.ReapedLeech, peer.HasIPv6())
			}
		}

		peers := torrent.PeerCount()
		c.torrentsM.Unlock()

		if purgeEmptyTorrents && peers == 0 {
			c.PurgeInactiveTorrent(infohash)
			stats.RecordEvent(stats.ReapedTorrent)
		}
	}

	return nil
}
