// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import "fmt"

var stringStoreDrivers = make(map[string]StringStoreDriver)

// StringStore represents an interface for manipulating strings.
type StringStore interface {
	// PutString adds the given string to the StringStore.
	PutString(s string) error

	// HasString returns whether or not the StringStore contains the given
	// string.
	HasString(s string) (bool, error)

	// RemoveString removes the string from the string store.
	// Returns ErrResourceDoesNotExist if the given string is not contained
	// in the store.
	RemoveString(s string) error
}

// StringStoreDriver represents an interface for creating a handle to the
// storage of strings.
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
		return nil, fmt.Errorf("store: unknown StringStoreDriver %q (forgotten import?)", cfg)
	}

	return driver.New(cfg)
}
