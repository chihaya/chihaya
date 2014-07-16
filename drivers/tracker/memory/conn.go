// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"time"

	"github.com/chihaya/chihaya/drivers/tracker"
	"github.com/chihaya/chihaya/models"
)

// Conn implements a connection to a memory-based tracker data store.
type Conn struct {
	*Pool
}

func (c *Conn) FindUser(passkey string) (*models.User, error) {
	c.usersM.RLock()
	defer c.usersM.RUnlock()

	user, exists := c.users[passkey]
	if !exists {
		return nil, tracker.ErrUserDNE
	}
	return &*user, nil
}

func (c *Conn) FindTorrent(infohash string) (*models.Torrent, error) {
	c.torrentsM.RLock()
	defer c.torrentsM.RUnlock()

	torrent, exists := c.torrents[infohash]
	if !exists {
		return nil, tracker.ErrTorrentDNE
	}
	return &*torrent, nil
}

func (c *Conn) FindClient(peerID string) error {
	c.whitelistM.RLock()
	defer c.whitelistM.RUnlock()

	_, ok := c.whitelist[peerID]
	if !ok {
		return tracker.ErrClientUnapproved
	}
	return nil
}

func (c *Conn) IncrementSnatches(infohash string) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}
	t.Snatches++

	return nil
}

func (c *Conn) TouchTorrent(infohash string) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}
	t.LastAction = time.Now().Unix()

	return nil
}

func (c *Conn) AddLeecher(infohash string, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}
	t.Leechers[p.Key()] = *p

	return nil
}

func (c *Conn) AddSeeder(infohash string, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}
	t.Seeders[p.Key()] = *p

	return nil
}

func (c *Conn) DeleteLeecher(infohash, peerkey string) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}
	delete(t.Leechers, peerkey)

	return nil
}

func (c *Conn) DeleteSeeder(infohash, peerkey string) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}
	delete(t.Seeders, peerkey)

	return nil
}

func (c *Conn) PutLeecher(infohash string, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}
	t.Leechers[p.Key()] = *p

	return nil
}

func (c *Conn) PutSeeder(infohash string, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	t, ok := c.torrents[infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}
	t.Seeders[p.Key()] = *p

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
