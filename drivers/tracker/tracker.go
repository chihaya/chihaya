// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package tracker provides a generic interface for manipulating a
// BitTorrent tracker's fast-moving data.
package tracker

import (
	"errors"
	"fmt"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/models"
)

var (
	// ErrUserDNE is returned when a user does not exist.
	ErrUserDNE = errors.New("user does not exist")
	// ErrTorrentDNE is returned when a torrent does not exist.
	ErrTorrentDNE = errors.New("torrent does not exist")
	// ErrClientUnapproved is returned when a clientID is not in the whitelist.
	ErrClientUnapproved = errors.New("client is not approved")
	// ErrInvalidPasskey is returned when a passkey is not properly formatted.
	ErrInvalidPasskey = errors.New("passkey is invalid")

	drivers = make(map[string]Driver)
)

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

// Open creates a pool of data store connections specified by a models configuration.
func Open(cfg *config.DriverConfig) (Pool, error) {
	driver, ok := drivers[cfg.Name]
	if !ok {
		return nil, fmt.Errorf(
			"unknown driver %q (forgotten import?)",
			cfg.Name,
		)
	}
	pool := driver.New(cfg)
	return pool, nil
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
	// Torrent interactions
	FindTorrent(infohash string) (*models.Torrent, error)
	PutTorrent(t *models.Torrent) error
	DeleteTorrent(infohash string) error
	IncrementSnatches(infohash string) error
	TouchTorrent(infohash string) error
	PutLeecher(infohash string, p *models.Peer) error
	DeleteLeecher(infohash, peerkey string) error
	PutSeeder(infohash string, p *models.Peer) error
	DeleteSeeder(infohash, peerkey string) error

	// User interactions
	FindUser(passkey string) (*models.User, error)
	PutUser(u *models.User) error
	DeleteUser(passkey string) error

	// Whitelist interactions
	FindClient(clientID string) error
	PutClient(clientID string) error
	DeleteClient(clientID string) error
}

// LeecherFinished moves a peer from the leeching pool to the seeder pool.
func LeecherFinished(c Conn, infohash string, p *models.Peer) error {
	err := c.DeleteLeecher(infohash, p.Key())
	if err != nil {
		return err
	}
	err = c.PutSeeder(infohash, p)
	return err
}
