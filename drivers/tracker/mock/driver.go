// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package mock implements the models interface for a BitTorrent tracker
// within memory. It can be used in production, but isn't recommended.
// Stored values will not persist if the tracker is restarted.
package mock

import (
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/drivers/tracker"
	"github.com/chihaya/chihaya/models"
)

type driver struct{}

func (d *driver) New(conf *config.DriverConfig) tracker.Pool {
	return &Pool{
		users:     make(map[string]*models.User),
		torrents:  make(map[string]*models.Torrent),
		whitelist: make(map[string]bool),
	}
}

func init() {
	tracker.Register("mock", &driver{})
}
