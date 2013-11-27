// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package mock implements the storage interface for a BitTorrent tracker
// within memory. It can be used in production, but isn't recommended.
// Stored values will not persist if the tracker is restarted.
package mock

import (
	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/storage"
	"github.com/pushrax/chihaya/storage/tracker"
)

type driver struct{}

func (d *driver) New(conf *config.DataStore) tracker.Pool {
	return &Pool{
		users:     make(map[string]*storage.User),
		torrents:  make(map[string]*storage.Torrent),
		whitelist: make(map[string]string),
	}
}
