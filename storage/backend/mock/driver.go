// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package mock implements the storage interface for a BitTorrent tracker's
// backend storage. It can be used in production, but isn't recommended.
// Stored values will not persist if the tracker is restarted.
package mock

import (
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/storage/backend"
)

type driver struct{}

func (d *driver) New(conf *config.DataStore) backend.Conn {
	return nil
}

func init() {
	backend.Register("mock", &driver{})
}
