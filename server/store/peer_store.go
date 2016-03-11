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
	PutSeeder(infoHash chihaya.InfoHash, p chihaya.Peer) error
	DeleteSeeder(infoHash chihaya.InfoHash, p chihaya.Peer) error

	PutLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error
	DeleteLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error

	GraduateLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error
	AnnouncePeers(infoHash chihaya.InfoHash, seeder bool, numWant int) (peers, peers6 []chihaya.Peer, err error)
	CollectGarbage(cutoff time.Time) error
}

// PeerStoreDriver represents an interface for creating a handle to the storage
// of peers.
type PeerStoreDriver interface {
	New(*DriverConfig) (PeerStore, error)
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
func OpenPeerStore(cfg *DriverConfig) (PeerStore, error) {
	driver, ok := peerStoreDrivers[cfg.Name]
	if !ok {
		return nil, fmt.Errorf("storage: unknown PeerStoreDriver %q (forgotten import?)", cfg)
	}

	return driver.New(cfg)
}
