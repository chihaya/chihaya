// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package backend provides a generic interface for manipulating a
// BitTorrent tracker's consistent backend data store (usually for
// a web application).
package backend

import (
	"fmt"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/models"
)

var drivers = make(map[string]Driver)

// Driver represents an interface to a long-running connection with a
// consistent data store.
type Driver interface {
	New(*config.DriverConfig) (Conn, error)
}

// Register makes a database driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver Driver) {
	if driver == nil {
		panic("backend: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("backend: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Open creates a connection specified by a models configuration.
func Open(cfg *config.DriverConfig) (Conn, error) {
	driver, ok := drivers[cfg.Name]
	if !ok {
		return nil, fmt.Errorf(
			"backend: unknown driver %q (forgotten import?)",
			cfg.Name,
		)
	}
	return driver.New(cfg)
}

// Conn represents a connection to the data store.
type Conn interface {
	// Close terminates connections to the database(s) and gracefully shuts
	// down the driver
	Close() error

	// RecordAnnounce is called once per announce, and is passed the delta in
	// statistics for the client peer since its last announce.
	RecordAnnounce(delta *models.AnnounceDelta) error

	// LoadTorrents fetches and returns the specified torrents.
	LoadTorrents(ids []uint64) ([]*models.Torrent, error)

	// LoadAllTorrents fetches and returns all torrents.
	LoadAllTorrents() ([]*models.Torrent, error)

	// LoadUsers fetches and returns the specified users.
	LoadUsers(ids []uint64) ([]*models.User, error)

	// LoadAllUsers fetches and returns all users.
	LoadAllUsers(ids []uint64) ([]*models.User, error)
}
