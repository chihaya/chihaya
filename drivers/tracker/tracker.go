// Copyright 2013 The Chihaya Authors. All rights reserved.
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
	New(*config.DataStore) Pool
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
func Open(conf *config.DataStore) (Pool, error) {
	driver, ok := drivers[conf.Driver]
	if !ok {
		return nil, fmt.Errorf(
			"unknown driver %q (forgotten import?)",
			conf.Driver,
		)
	}
	pool := driver.New(conf)
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
	// Reads
	FindUser(passkey string) (*models.User, error)
	FindTorrent(infohash string) (*models.Torrent, error)
	ClientWhitelisted(clientID string) error

	// Writes
	IncrementSnatches(t *models.Torrent) error
	MarkActive(t *models.Torrent) error
	AddLeecher(t *models.Torrent, p *models.Peer) error
	AddSeeder(t *models.Torrent, p *models.Peer) error
	RemoveLeecher(t *models.Torrent, p *models.Peer) error
	RemoveSeeder(t *models.Torrent, p *models.Peer) error
	SetLeecher(t *models.Torrent, p *models.Peer) error
	SetSeeder(t *models.Torrent, p *models.Peer) error

	// Priming / Testing
	AddTorrent(t *models.Torrent) error
	RemoveTorrent(t *models.Torrent) error
	AddUser(u *models.User) error
	RemoveUser(u *models.User) error
	WhitelistClient(clientID string) error
	UnWhitelistClient(clientID string) error
}

// LeecherFinished moves a peer from the leeching pool to the seeder pool.
func LeecherFinished(c Conn, t *models.Torrent, p *models.Peer) error {
	err := c.RemoveLeecher(t, p)
	if err != nil {
		return err
	}
	err = c.AddSeeder(t, p)
	return err
}
