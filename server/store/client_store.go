// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"fmt"
	"github.com/chihaya/chihaya"
)

var clientStoreDrivers = make(map[string]ClientStoreDriver)

// ClientStore represents an interface for manipulating clientIDs.
type ClientStore interface {
	CreateClient(clientID string) error
	FindClient(peerID chihaya.PeerID) (bool, error)
	DeleteClient(clientID string) error
}

// ClientStoreDriver represents an interface for creating a handle to the
// storage of swarms.
type ClientStoreDriver interface {
	New(*DriverConfig) (ClientStore, error)
}

// RegisterClientStoreDriver makes a driver available by the provided name.
//
// If this function is called twice with the same name or if the driver is nil,
// it panics.
func RegisterClientStoreDriver(name string, driver ClientStoreDriver) {
	if driver == nil {
		panic("store: could not register nil ClientStoreDriver")
	}
	if _, dup := clientStoreDrivers[name]; dup {
		panic("store: could not register duplicate ClientStoreDriver: " + name)
	}
	clientStoreDrivers[name] = driver
}

// OpenClientStore returns a ClientStore specified by a configuration.
func OpenClientStore(cfg *DriverConfig) (ClientStore, error) {
	driver, ok := clientStoreDrivers[cfg.Name]
	if !ok {
		return nil, fmt.Errorf("store: unknown driver %q (forgotten import?)", cfg)
	}

	return driver.New(cfg)
}
