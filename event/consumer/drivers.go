// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package consumer provides a generic interface for consuming events emitted
// by a BitTorrent tracker.
package consumer

import (
	"fmt"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker/models"
)

var drivers = make(map[string]Driver)

// Driver represents a dyanmic way to instantiate a Consumer.
type Driver interface {
	New(*config.DriverConfig) (Consumer, error)
}

// Register provides a way to dynamically register an implementation of a
// Consumer as a driver.
//
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver Driver) {
	if driver == nil {
		panic("consumer: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("consumer: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Open creates a Consumer specified by a configuration.
func Open(cfg *config.DriverConfig) (Consumer, error) {
	driver, ok := drivers[cfg.Name]
	if !ok {
		return nil, fmt.Errorf(
			"consumer: unknown driver %q (forgotten import?)",
			cfg.Name,
		)
	}
	return driver.New(cfg)
}

// Consumer represents something that can consume events from a BitTorrent
// tracker.
type Consumer interface {
	// Close terminates connections to the database(s) and gracefully shuts
	// down the driver
	Close() error

	// RecordAnnounce is called once per announce, and is passed the delta in
	// statistics for the client peer since its last announce.
	RecordAnnounce(delta *models.AnnounceDelta) error
}
