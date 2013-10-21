// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package backend provides a generic interface for manipulating a
// BitTorrent tracker's backend data (usually for a web application).
package backend

import (
	"fmt"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/storage"
)

var drivers = make(map[string]Driver)

type Driver interface {
	New(*config.DataStore) Conn
}

// Register makes a database driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver Driver) {
	if driver == nil {
		panic("web: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("web: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Open creates a connection specified by a storage configuration.
func Open(conf *config.DataStore) (Conn, error) {
	driver, ok := drivers[conf.Driver]
	if !ok {
		return nil, fmt.Errorf(
			"web: unknown driver %q (forgotten import?)",
			conf.Driver,
		)
	}
	pool := driver.New(conf)
	return pool, nil
}

// Conn represents a connection to the data store.
type Conn interface {
	// Start is called once when the server starts.
	// It starts any necessary goroutines a given driver requires, and sets
	// up the driver's initial state
	Start() error

	// Close terminates connections to the database(s) and gracefully shuts
	// down the driver
	Close() error

	// RecordAnnounce is called once per announce, and is passed the delta in
	// statistics for the client peer since its last announce.
	RecordAnnounce(delta *AnnounceDelta) error

	// LoadTorrents fetches and returns the specified torrents.
	LoadTorrents(ids []uint64) ([]*storage.Torrent, error)

	// LoadAllTorrents fetches and returns all torrents.
	LoadAllTorrents() ([]*storage.Torrent, error)

	// LoadUsers fetches and returns the specified users.
	LoadUsers(ids []uint64) ([]*storage.User, error)

	// LoadAllUsers fetches and returns all users.
	LoadAllUsers(ids []uint64) ([]*storage.User, error)
}

// AnnounceDelta contains a difference in statistics for a peer.
// It is used for communicating changes to be recorded by the driver.
type AnnounceDelta struct {
	Peer    *storage.Peer
	Torrent *storage.Torrent
	User    *storage.User

	// Created is true if this announce created a new peer or changed an existing peer's address
	Created bool

	// Uploaded contains the raw upload delta for this announce, in bytes
	Uploaded uint64

	// Downloaded contains the raw download delta for this announce, in bytes
	Downloaded uint64

	// Timestamp is the unix timestamp this announce occurred at
	Timestamp float64

	// Snatched is true if this announce completed the download
	Snatched bool
}
