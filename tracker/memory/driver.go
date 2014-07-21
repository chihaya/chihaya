// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package memory implements a Chihaya tracker storage driver within memory.
// Stored values will not persist if the tracker is restarted.
package memory

import (
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker"
	"github.com/chihaya/chihaya/tracker/models"
)

type driver struct{}

func (d *driver) New(cfg *config.DriverConfig) tracker.Pool {
	return &Pool{
		users:     make(map[string]*models.User),
		torrents:  make(map[string]*models.Torrent),
		whitelist: make(map[string]bool),
	}
}

func init() {
	tracker.Register("memory", &driver{})
}
