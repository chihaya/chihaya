// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import "fmt"

// PrefixInfohash is the prefix to be used for infohashes.
const PrefixInfohash = "ih-"

var stringStoreDrivers = make(map[string]StringStoreDriver)

// StringStore represents an interface for manipulating strings.
type StringStore interface {
	PutString(s string) error
	HasString(s string) (bool, error)
	RemoveString(s string) error
}

// StringStoreDriver represents an interface for creating a handle to the
// storage of swarms.
type StringStoreDriver interface {
	New(*DriverConfig) (StringStore, error)
}

// RegisterStringStoreDriver makes a driver available by the provided name.
//
// If this function is called twice with the same name or if the driver is nil,
// it panics.
func RegisterStringStoreDriver(name string, driver StringStoreDriver) {
	if driver == nil {
		panic("store: could not register nil StringStoreDriver")
	}
	if _, dup := stringStoreDrivers[name]; dup {
		panic("store: could not register duplicate StringStoreDriver: " + name)
	}
	stringStoreDrivers[name] = driver
}

// OpenStringStore returns a StringStore specified by a configuration.
func OpenStringStore(cfg *DriverConfig) (StringStore, error) {
	driver, ok := stringStoreDrivers[cfg.Name]
	if !ok {
		return nil, fmt.Errorf("store: unknown driver %q (forgotten import?)", cfg)
	}

	return driver.New(cfg)
}
