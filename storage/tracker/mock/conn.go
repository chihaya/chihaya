// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package mock

import (
	"github.com/pushrax/chihaya/storage"
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
	return user, true, nil
}

func (c *Conn) FindTorrent(infohash string) (*storage.Torrent, bool, error) {
	c.torrentsM.RLock()
	defer c.torrentsM.RUnlock()
	torrent, ok := c.torrents[infohash]
	if !ok {
		return nil, false, nil
	}
	return torrent, true, nil
}

func (c *Conn) ClientWhitelisted(peerID string) (bool, error) {
	_, ok := c.whitelist[peerID]
	if !ok {
		return false, nil
	}
	return true, nil
}

func (c *Conn) RecordSnatch(u *storage.User, t *storage.Torrent) error {
	return nil
}

func (c *Conn) MarkActive(t *storage.Torrent) error {
	return nil
}

func (c *Conn) AddLeecher(t *storage.Torrent, p *storage.Peer) error {
	return nil
}

func (c *Conn) AddSeeder(t *storage.Torrent, p *storage.Peer) error {
	return nil
}

func (c *Conn) RemoveLeecher(t *storage.Torrent, p *storage.Peer) error {
	return nil
}

func (c *Conn) RemoveSeeder(t *storage.Torrent, p *storage.Peer) error {
	return nil
}

func (c *Conn) SetLeecher(t *storage.Torrent, p *storage.Peer) error {
	return nil
}

func (c *Conn) SetSeeder(t *storage.Torrent, p *storage.Peer) error {
	return nil
}

func (c *Conn) IncrementSlots(u *storage.User) error {
	return nil
}

func (c *Conn) DecrementSlots(u *storage.User) error {
	return nil
}

func (c *Conn) LeecherFinished(t *storage.Torrent, p *storage.Peer) error {
	return nil
}

func (c *Conn) AddTorrent(t *storage.Torrent) error {
	return nil
}

func (c *Conn) RemoveTorrent(t *storage.Torrent) error {
	return nil
}

func (c *Conn) AddUser(u *storage.User) error {
	return nil
}

func (c *Conn) RemoveUser(u *storage.User) error {
	return nil
}

func (c *Conn) WhitelistClient(peerID string) error {
	return nil
}

func (c *Conn) UnWhitelistClient(peerID string) error {
	return nil
}
