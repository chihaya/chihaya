// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"fmt"
	"time"

	"github.com/chihaya/chihaya"
)

var peerStoreDrivers = make(map[string]PeerStoreDriver)

// PeerStore represents an interface for manipulating peers.
type PeerStore interface {
	PutSeeder(infohash string, p chihaya.Peer) error
	DeleteSeeder(infohash, peerID string) error

	PutLeecher(infohash string, p chihaya.Peer) error
	DeleteLeecher(infohash, peerID string) error

	GraduateLeecher(infohash string, p chihaya.Peer) error
	AnnouncePeers(infohash string, seeder bool, numWant int) (peers, peers6 []chihaya.Peer, err error)
	CollectGarbage(cutoff time.Time) error
}

// PeerStoreDriver represents an interface for creating a handle to the storage
// of peers.
type PeerStoreDriver interface {
	New(*Config) (PeerStore, error)
}

// RegisterPeerStoreDriver makes a driver available by the provided name.
//
// If this function is called twice with the same name or if the driver is nil,
// it panics.
func RegisterPeerStoreDriver(name string, driver PeerStoreDriver) {
	if driver == nil {
		panic("storage: could not register nil PeerStoreDriver")
	}

	if _, dup := peerStoreDrivers[name]; dup {
		panic("storage: could not register duplicate PeerStoreDriver: " + name)
	}

	peerStoreDrivers[name] = driver
}

// OpenPeerStore returns a PeerStore specified by a configuration.
func OpenPeerStore(name string, cfg *Config) (PeerStore, error) {
	driver, ok := peerStoreDrivers[name]
	if !ok {
		return nil, fmt.Errorf(
			"storage: unknown driver %q (forgotten import?)",
			name,
		)
	}

	return driver.New(cfg)
}
