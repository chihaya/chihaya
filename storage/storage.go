// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package storage provides a generic interface for manipulating a
// BitTorrent tracker's web application data.
package storage

import (
	"fmt"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/models"
)

var (
	drivers = make(map[string]Driver)
)

type Driver interface {
	New(*config.DataStore) Conn
}

// Register makes a database driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver Driver) {
	if driver == nil {
		panic("storage: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("storage: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Open creates a connection specified by a storage configuration.
func Open(conf *config.DataStore) (Conn, error) {
	driver, ok := drivers[conf.Driver]
	if !ok {
		return nil, fmt.Errorf(
			"storage: unknown driver %q (forgotten import?)",
			conf.Driver,
		)
	}
	pool := driver.New(conf)
	return pool, nil
}

// Conn represents a connection to the data store.
type Conn interface {
	Start() error
	Close() error
	RecordAnnounce(delta *models.AnnounceDelta) error
	RecordSnatch(peer *models.Peer) error
}
