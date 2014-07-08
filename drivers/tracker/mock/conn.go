// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package mock

import (
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

func (c *Conn) IncrementSnatches(t *models.Torrent) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}
	torrent.Snatches++
	t.Snatches++

	return nil
}

func (c *Conn) MarkActive(t *models.Torrent) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}

	torrent.Active = true
	t.Active = true

	return nil
}

func (c *Conn) MarkInactive(t *models.Torrent) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}

	torrent.Active = false
	t.Active = false

	return nil
}

func (c *Conn) AddLeecher(t *models.Torrent, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}

	torrent.Leechers[p.Key()] = *p
	t.Leechers[p.Key()] = *p

	return nil
}

func (c *Conn) AddSeeder(t *models.Torrent, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}

	torrent.Seeders[p.Key()] = *p
	t.Seeders[p.Key()] = *p

	return nil
}

func (c *Conn) RemoveLeecher(t *models.Torrent, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}

	delete(torrent.Leechers, p.Key())
	delete(t.Leechers, p.Key())

	return nil
}

func (c *Conn) RemoveSeeder(t *models.Torrent, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}

	delete(torrent.Seeders, p.Key())
	delete(t.Seeders, p.Key())

	return nil
}

func (c *Conn) SetLeecher(t *models.Torrent, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}

	torrent.Leechers[p.Key()] = *p
	t.Leechers[p.Key()] = *p

	return nil
}

func (c *Conn) SetSeeder(t *models.Torrent, p *models.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrTorrentDNE
	}

	torrent.Seeders[p.Key()] = *p
	t.Seeders[p.Key()] = *p

	return nil
}

func (c *Conn) AddTorrent(t *models.Torrent) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent := *t
	c.torrents[t.Infohash] = &torrent

	return nil
}

func (c *Conn) RemoveTorrent(infohash string) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	delete(c.torrents, infohash)

	return nil
}

func (c *Conn) AddUser(u *models.User) error {
	c.usersM.Lock()
	defer c.usersM.Unlock()

	user := *u
	c.users[u.Passkey] = &user

	return nil
}

func (c *Conn) RemoveUser(passkey string) error {
	c.usersM.Lock()
	defer c.usersM.Unlock()

	delete(c.users, passkey)

	return nil
}

func (c *Conn) AddClient(peerID string) error {
	c.whitelistM.Lock()
	defer c.whitelistM.Unlock()

	c.whitelist[peerID] = true

	return nil
}

func (c *Conn) RemoveClient(peerID string) error {
	c.whitelistM.Lock()
	defer c.whitelistM.Unlock()

	delete(c.whitelist, peerID)

	return nil
}
