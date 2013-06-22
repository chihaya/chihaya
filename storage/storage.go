// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package storage

import (
	"fmt"

	"github.com/pushrax/chihaya/config"
)

var drivers = make(map[string]StorageDriver)

type StorageDriver interface {
	New(*config.Storage) (Storage, error)
}

func Register(name string, driver StorageDriver) {
	if driver == nil {
		panic("storage: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("storage: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

func New(conf *config.Storage) (Storage, error) {
	driver, ok := drivers[conf.Driver]
	if !ok {
		return nil, fmt.Errorf(
			"storage: unknown driver %q (forgotten import?)",
			conf.Driver,
		)
	}
	store, err := driver.New(conf)
	if err != nil {
		return nil, err
	}
	return store, nil
}

type Storage interface {
	Shutdown() error

	FindUser(passkey string) (*User, bool, error)
	FindTorrent(infohash string) (*Torrent, bool, error)
	UnpruneTorrent(torrent *Torrent) error

	RecordUser(
		user *User,
		rawDeltaUpload int64,
		rawDeltaDownload int64,
		deltaUpload int64,
		deltaDownload int64,
	) error
	RecordSnatch(peer *Peer, now int64) error
	RecordTorrent(torrent *Torrent, deltaSnatch uint64) error
	RecordTransferIP(peer *Peer) error
	RecordTransferHistory(
		peer *Peer,
		rawDeltaUpload int64,
		rawDeltaDownload int64,
		deltaTime int64,
		deltaSnatch uint64,
		active bool,
	) error
}
