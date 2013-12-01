// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package gazelle

import (
	"github.com/chihaya/chihaya/storage"
)

func (c *Conn) LoadTorrents(ids []uint64) ([]*storage.Torrent, error) {
	return nil, nil
}

func (c *Conn) LoadAllTorrents() ([]*storage.Torrent, error) {
	return nil, nil
}

func (c *Conn) LoadUsers(ids []uint64) ([]*storage.User, error) {
	return nil, nil
}

func (c *Conn) LoadAllUsers(ids []uint64) ([]*storage.User, error) {
	return nil, nil
}
