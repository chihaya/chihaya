// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"fmt"
	"time"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker/models"
)

var drivers = make(map[string]Driver)

// Driver represents an interface to pool of connections to models used for
// the tracker.
type Driver interface {
	New(*config.DriverConfig) Pool
}

// Register makes a database driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver Driver) {
	if driver == nil {
		panic("tracker: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("tracker: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Open creates a pool of data store connections specified by a configuration.
func Open(cfg *config.DriverConfig) (Pool, error) {
	driver, ok := drivers[cfg.Name]
	if !ok {
		return nil, fmt.Errorf(
			"unknown driver %q (forgotten import?)",
			cfg.Name,
		)
	}
	return driver.New(cfg), nil
}

// Pool represents a thread-safe pool of connections to the data store
// that can be used to safely within concurrent goroutines.
type Pool interface {
	Close() error
	Get() (Conn, error)
}

// Conn represents a connection to the data store that can be used
// to make reads/writes.
type Conn interface {
	Close() error

	// Torrent interactions
	TouchTorrent(infohash string) error
	FindTorrent(infohash string) (*models.Torrent, error)
	PutTorrent(t *models.Torrent) error
	DeleteTorrent(infohash string) error
	IncrementTorrentSnatches(infohash string) error

	PutLeecher(infohash, ipv string, p *models.Peer) error
	DeleteLeecher(infohash string, pk models.PeerKey) error

	PutSeeder(infohash, ipv string, p *models.Peer) error
	DeleteSeeder(infohash string, pk models.PeerKey) error

	PurgeInactiveTorrent(infohash string) error
	PurgeInactivePeers(purgeEmptyTorrents bool, before time.Time) error

	// User interactions
	FindUser(passkey string) (*models.User, error)
	PutUser(u *models.User) error
	DeleteUser(passkey string) error
	IncrementUserSnatches(passkey string) error

	// Whitelist interactions
	FindClient(clientID string) error
	PutClient(clientID string) error
	DeleteClient(clientID string) error
}
