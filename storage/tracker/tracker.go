// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package tracker provides a generic interface for manipulating a
// BitTorrent tracker's fast-moving, inconsistent data.
package tracker

import (
	"fmt"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/storage"
)

var (
	drivers = make(map[string]Driver)
)

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

// Open creates a pool of data store connections specified by a storage configuration.
func Open(conf *config.DataStore) (Pool, error) {
	driver, ok := drivers[conf.Driver]
	if !ok {
		return nil, fmt.Errorf(
			"tracker: unknown driver %q (forgotten import?)",
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
// to make atomic and non-atomic reads/writes.
type Conn interface {
	// Reads
	FindUser(passkey string) (*storage.User, bool, error)
	FindTorrent(infohash string) (*storage.Torrent, bool, error)
	ClientWhitelisted(peerID string) (bool, error)

	// Writes
	RecordSnatch(u *storage.User, t *storage.Torrent) error
	MarkActive(t *storage.Torrent) error
	AddLeecher(t *storage.Torrent, p *storage.Peer) error
	AddSeeder(t *storage.Torrent, p *storage.Peer) error
	RemoveLeecher(t *storage.Torrent, p *storage.Peer) error
	RemoveSeeder(t *storage.Torrent, p *storage.Peer) error
	SetLeecher(t *storage.Torrent, p *storage.Peer) error
	SetSeeder(t *storage.Torrent, p *storage.Peer) error
	IncrementSlots(u *storage.User) error
	DecrementSlots(u *storage.User) error
	LeecherFinished(t *storage.Torrent, p *storage.Peer) error

	// Priming / Testing
	AddTorrent(t *storage.Torrent) error
	RemoveTorrent(t *storage.Torrent) error
	AddUser(u *storage.User) error
	RemoveUser(u *storage.User) error
	WhitelistClient(peerID string) error
	UnWhitelistClient(peerID string) error
}
