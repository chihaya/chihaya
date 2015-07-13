// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package store provides a generic interface for manipulating the primary
// storage system for a BitTorrent tracker.
package store

import (
	"fmt"
	"time"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker/models"
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
		panic("store: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("store: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Open creates an implementation of Conn specified by a configuration.
func Open(cfg *config.DriverConfig) (Conn, error) {
	driver, ok := drivers[cfg.Name]
	if !ok {
		return nil, fmt.Errorf(
			"unknown driver %q (forgotten import?)",
			cfg.Name,
		)
	}

	return driver.New(cfg)
}

// Conn represents a connection (or pool of connections) to the data store.
type Conn interface {
	// Close terminates connections to the database(s) and gracefully shuts
	// down the driver
	Close() error

	// Torrent manipulations
	TouchTorrent(infohash string) error
	FindTorrent(infohash string) (*models.Torrent, error)
	PutTorrent(t *models.Torrent) error
	DeleteTorrent(infohash string) error
	IncrementSnatches(infohash string) error
	PutLeecher(infohash string, peer *models.Peer) error
	DeleteLeecher(infohash string, pk models.PeerKey) error
	PutSeeder(infohash string, ppeer *models.Peer) error
	DeleteSeeder(infohash string, pk models.PeerKey) error

	PurgeInactiveTorrent(infohash string) error
	PurgeInactivePeers(purgeEmptyTorrents bool, before time.Time) error

	// User manipulations
	FindUser(passkey string) (*models.User, error)
	PutUser(user *models.User) error
	DeleteUser(passkey string) error

	// Whitelist manipulations
	FindClient(clientID string) error
	PutClient(clientID string) error
	DeleteClient(clientID string) error
}
