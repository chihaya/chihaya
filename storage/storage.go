// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package storage provides a generic interface for manipulating a
// BitTorrent tracker's data store.
package storage

import (
	"errors"
	"fmt"

	"github.com/pushrax/chihaya/config"
)

var (
	drivers   = make(map[string]Driver)
	ErrTxDone = errors.New("storage: Transaction has already been committed or rolled back")
)

type Driver interface {
	New(*config.Storage) Pool
}

// Register makes a database driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver Driver) {
	if driver == nil {
		panic("storage: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("storage: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Open creates a pool of data store connections specified by a storage configuration.
func Open(conf *config.Storage) (Pool, error) {
	driver, ok := drivers[conf.Driver]
	if !ok {
		return nil, fmt.Errorf(
			"storage: unknown driver %q (forgotten import?)",
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
	FindUser(passkey string) (*User, bool, error)
	FindTorrent(infohash string) (*Torrent, bool, error)
	ClientWhitelisted(peerID string) (bool, error)

	// Writes
	Snatch(u *User, t *Torrent) error
	MarkActive(t *Torrent) error
	NewLeecher(t *Torrent, p *Peer) error
	NewSeeder(t *Torrent, p *Peer) error
	RmLeecher(t *Torrent, p *Peer) error
	RmSeeder(t *Torrent, p *Peer) error
	SetLeecher(t *Torrent, p *Peer) error
	SetSeeder(t *Torrent, p *Peer) error
	IncrementSlots(u *User) error
	DecrementSlots(u *User) error
}
