// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package mock

import (
	"github.com/chihaya/chihaya/storage"
	"github.com/chihaya/chihaya/storage/tracker"
)

type Conn struct {
	*Pool
}

func (c *Conn) FindUser(passkey string) (*storage.User, bool, error) {
	c.usersM.RLock()
	defer c.usersM.RUnlock()
	user, ok := c.users[passkey]
	if !ok {
		return nil, false, nil
	}
	u := *user
	return &u, true, nil
}

func (c *Conn) FindTorrent(infohash string) (*storage.Torrent, bool, error) {
	c.torrentsM.RLock()
	defer c.torrentsM.RUnlock()
	torrent, ok := c.torrents[infohash]
	if !ok {
		return nil, false, nil
	}
	t := *torrent
	return &t, true, nil
}

func (c *Conn) ClientWhitelisted(peerID string) (bool, error) {
	c.whitelistM.RLock()
	defer c.whitelistM.RUnlock()
	_, ok := c.whitelist[peerID]
	if !ok {
		return false, nil
	}
	return true, nil
}

func (c *Conn) RecordSnatch(u *storage.User, t *storage.Torrent) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()
	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrMissingResource
	}
	torrent.Snatches++
	t.Snatches++
	return nil
}

func (c *Conn) MarkActive(t *storage.Torrent) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()
	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrMissingResource
	}
	torrent.Active = true
	t.Active = true
	return nil
}

func (c *Conn) MarkInactive(t *storage.Torrent) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()
	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrMissingResource
	}
	torrent.Active = false
	t.Active = false
	return nil
}

func (c *Conn) AddLeecher(t *storage.Torrent, p *storage.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()
	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrMissingResource
	}
	torrent.Leechers[storage.PeerMapKey(p)] = *p
	t.Leechers[storage.PeerMapKey(p)] = *p
	return nil
}

func (c *Conn) AddSeeder(t *storage.Torrent, p *storage.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()
	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrMissingResource
	}
	torrent.Leechers[storage.PeerMapKey(p)] = *p
	t.Leechers[storage.PeerMapKey(p)] = *p
	return nil
}

func (c *Conn) RemoveLeecher(t *storage.Torrent, p *storage.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()
	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrMissingResource
	}
	delete(torrent.Leechers, storage.PeerMapKey(p))
	delete(t.Leechers, storage.PeerMapKey(p))
	return nil
}

func (c *Conn) RemoveSeeder(t *storage.Torrent, p *storage.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()
	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrMissingResource
	}
	delete(torrent.Seeders, storage.PeerMapKey(p))
	delete(t.Seeders, storage.PeerMapKey(p))
	return nil
}

func (c *Conn) SetLeecher(t *storage.Torrent, p *storage.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()
	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrMissingResource
	}
	torrent.Leechers[storage.PeerMapKey(p)] = *p
	t.Leechers[storage.PeerMapKey(p)] = *p
	return nil
}

func (c *Conn) SetSeeder(t *storage.Torrent, p *storage.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()
	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrMissingResource
	}
	torrent.Seeders[storage.PeerMapKey(p)] = *p
	t.Seeders[storage.PeerMapKey(p)] = *p
	return nil
}

func (c *Conn) LeecherFinished(t *storage.Torrent, p *storage.Peer) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()

	torrent, ok := c.torrents[t.Infohash]
	if !ok {
		return tracker.ErrMissingResource
	}

	torrent.Seeders[storage.PeerMapKey(p)] = *p
	delete(torrent.Leechers, storage.PeerMapKey(p))
	t.Seeders[storage.PeerMapKey(p)] = *p
	delete(t.Leechers, storage.PeerMapKey(p))
	return nil
}

func (c *Conn) AddTorrent(t *storage.Torrent) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()
	torrent := *t
	c.torrents[t.Infohash] = &torrent
	return nil
}

func (c *Conn) RemoveTorrent(t *storage.Torrent) error {
	c.torrentsM.Lock()
	defer c.torrentsM.Unlock()
	delete(c.torrents, t.Infohash)
	return nil
}

func (c *Conn) AddUser(u *storage.User) error {
	c.usersM.Lock()
	defer c.usersM.Unlock()
	user := *u
	c.users[u.Passkey] = &user
	return nil
}

func (c *Conn) RemoveUser(u *storage.User) error {
	c.usersM.Lock()
	defer c.usersM.Unlock()
	delete(c.users, u.Passkey)
	return nil
}

func (c *Conn) WhitelistClient(peerID string) error {
	c.whitelistM.Lock()
	defer c.whitelistM.Unlock()
	c.whitelist[peerID] = true
	return nil
}

func (c *Conn) UnWhitelistClient(peerID string) error {
	c.whitelistM.Lock()
	defer c.whitelistM.Unlock()
	delete(c.whitelist, peerID)
	return nil
}
