// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package cache provides a generic interface for manipulating a
// BitTorrent tracker's fast moving data.
package cache

import (
	"errors"
	"fmt"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/models"
)

var (
	drivers       = make(map[string]Driver)
	ErrTxDone     = errors.New("cache: Transaction has already been committed or rolled back")
	ErrTxConflict = errors.New("cache: Commit interrupted, update transaction and repeat")
)

type Driver interface {
	New(*config.DataStore) Pool
}

// Register makes a database driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver Driver) {
	if driver == nil {
		panic("cache: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("cache: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Open creates a pool of data store connections specified by a storage configuration.
func Open(conf *config.DataStore) (Pool, error) {
	driver, ok := drivers[conf.Driver]
	if !ok {
		return nil, fmt.Errorf(
			"cache: unknown driver %q (forgotten import?)",
			conf.Driver,
		)
	}
	pool := driver.New(conf)
	return pool, nil
}

// Pool represents a thread-safe pool of connections to the data store
// that can be used to obtain transactions.
type Pool interface {
	Close() error
	Get() (Tx, error)
}

// Tx represents an in-progress data store transaction.
// A transaction must end with a call to Commit or Rollback.
//
// After a call to Commit or Rollback, all operations on the
// transaction must fail with ErrTxDone.
type Tx interface {
	Commit() error
	Rollback() error

	// Reads
	FindUser(passkey string) (*models.User, bool, error)
	FindTorrent(infohash string) (*models.Torrent, bool, error)
	ClientWhitelisted(peerID string) (bool, error)

	// Writes
	RecordSnatch(u *models.User, t *models.Torrent) error
	MarkActive(t *models.Torrent) error
	AddLeecher(t *models.Torrent, p *models.Peer) error
	AddSeeder(t *models.Torrent, p *models.Peer) error
	RemoveLeecher(t *models.Torrent, p *models.Peer) error
	RemoveSeeder(t *models.Torrent, p *models.Peer) error
	SetLeecher(t *models.Torrent, p *models.Peer) error
	SetSeeder(t *models.Torrent, p *models.Peer) error
	IncrementSlots(u *models.User) error
	DecrementSlots(u *models.User) error
}
