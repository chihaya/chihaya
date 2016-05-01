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
	// PutSeeder adds a seeder for the infoHash to the PeerStore.
	PutSeeder(infoHash chihaya.InfoHash, p chihaya.Peer) error
	// DeleteSeeder removes a seeder for the infoHash from the PeerStore.
	//
	// Returns ErrResourceDoesNotExist if the infoHash or peer does not
	// exist.
	DeleteSeeder(infoHash chihaya.InfoHash, p chihaya.Peer) error

	// PutLeecher adds a leecher for the infoHash to the PeerStore.
	PutLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error
	// DeleteLeecher removes a leecher for the infoHash from the PeerStore.
	//
	// Returns ErrResourceDoesNotExist if the infoHash or peer does not
	// exist.
	DeleteLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error

	// GraduateLeecher promotes a peer from a leecher to a seeder for the
	// infoHash within the PeerStore.
	GraduateLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error
	// AnnouncePeers returns a list of both IPv4, and IPv6 peers for an
	// announce.
	//
	// If seeder is true then the peers returned will only be leechers, the
	// ammount of leechers returned will be the smaller value of numWant or
	// the available leechers.
	// If it is false then seeders will be returned up until numWant or the
	// available seeders, whichever is smaller. If the available seeders is
	// less than numWant then peers are returned until numWant or they run out.
	AnnouncePeers(infoHash chihaya.InfoHash, seeder bool, numWant int, peer4, peer6 chihaya.Peer) (peers, peers6 []chihaya.Peer, err error)
	// CollectGarbage deletes peers from the peerStore which are older than the
	// cutoff time.
	CollectGarbage(cutoff time.Time) error

	// GetSeeders gets all the seeders for a particular infoHash.
	GetSeeders(infoHash chihaya.InfoHash) (peers, peers6 []chihaya.Peer, err error)
	// GetLeechers gets all the leechers for a particular infoHash.
	GetLeechers(infoHash chihaya.InfoHash) (peers, peers6 []chihaya.Peer, err error)

	// NumSeeders gets the amount of seeders for a particular infoHash.
	NumSeeders(infoHash chihaya.InfoHash) int
	// NumLeechers gets the amount of leechers for a particular infoHash.
	NumLeechers(infoHash chihaya.InfoHash) int
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
