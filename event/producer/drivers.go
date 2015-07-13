// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package producer provides a generic interface for producing events to be
// consumed by a BitTorrent tracker.
package producer

import (
	"fmt"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker/models"
)

var drivers = make(map[string]Driver)

// Driver represents a dynamic way to instantiate a Producer.
type Driver interface {
	New(*config.DriverConfig) (Producer, error)
}

// Register provides a way to dynamically register an implementation of a
// Producer as a driver.
//
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver Driver) {
	if driver == nil {
		panic("producer: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("producer: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Open creates a Producer specified by a configuration.
func Open(cfg *config.DriverConfig) (Producer, error) {
	driver, ok := drivers[cfg.Name]
	if !ok {
		return nil, fmt.Errorf(
			"producer: unknown driver %q (forgotten import?)",
			cfg.Name,
		)
	}
	return driver.New(cfg)
}

// Producer represents something that can emit events to be consumed by a
// BitTorrent tracker.
type Producer interface {
	// Close terminates connections to the database(s) and gracefully shuts
	// down the driver
	Close() error

	// LoadTorrents fetches and returns the specified torrents.
	LoadTorrents(ids []uint64) ([]*models.Torrent, error)

	// LoadAllTorrents fetches and returns all torrents.
	LoadAllTorrents() ([]*models.Torrent, error)

	// LoadUsers fetches and returns the specified users.
	LoadUsers(ids []uint64) ([]*models.User, error)

	// LoadAllUsers fetches and returns all users.
	LoadAllUsers(ids []uint64) ([]*models.User, error)
}
