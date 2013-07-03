// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package storage provides a generic interface for manipulating a
// BitTorrent tracker's data store.
package storage

import (
	"fmt"

	"github.com/pushrax/chihaya/config"
)

var drivers = make(map[string]Driver)

type Driver interface {
	New(*config.Storage) Pool
}

func Register(name string, driver Driver) {
	if driver == nil {
		panic("storage: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("storage: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

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

// ConnPool represents a pool of connections to the data store.
type Pool interface {
	Close() error
	Get() Conn
}

// Conn represents a single connection to the data store.
type Conn interface {
	Close() error

	NewTx() (Tx, error)

	FindUser(passkey string) (*User, bool, error)
	FindTorrent(infohash string) (*Torrent, bool, error)
}

// Tx represents a data store transaction.
type Tx interface {
	Commit() error

	UnpruneTorrent(torrent *Torrent) error
}
