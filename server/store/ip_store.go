// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"fmt"
	"net"
)

var ipStoreDrivers = make(map[string]IPStoreDriver)

// IPStore represents an interface for manipulating IPs and IP ranges.
type IPStore interface {
	// AddIP adds a single IP address to the IPStore.
	AddIP(ip net.IP) error

	// AddNetwork adds a range of IP addresses, denoted by a network in CIDR
	// notation, to the IPStore.
	AddNetwork(network string) error

	// HasIP returns whether the given IP address is contained in the IPStore
	// or belong to any of the stored networks.
	HasIP(ip net.IP) (bool, error)

	// HasAnyIP returns whether any of the given IP addresses are contained in
	// the IPStore or belong to any of the stored networks.
	HasAnyIP(ips []net.IP) (bool, error)

	// HassAllIPs returns whether all of the given IP addresses are contained in
	// the IPStore or belong to any of the stored networks.
	HasAllIPs(ips []net.IP) (bool, error)

	// RemoveIP removes a single IP address from the IPStore.
	//
	// This wil not remove the given address from any networks it belongs to
	// that are stored in the IPStore.
	RemoveIP(ip net.IP) error

	// RemoveNetwork removes a range of IP addresses that was previously added
	// through AddNetwork.
	//
	// The given network must not, as a string, match the previously added
	// network, but rather denote the same network, e.g. if the network
	// 192.168.22.255/24 was added, removing the network 192.168.22.123/24
	// will succeed.
	RemoveNetwork(network string) error
}

// IPStoreDriver represents an interface for creating a handle to the
// storage of IPs.
type IPStoreDriver interface {
	New(*Config) (IPStore, error)
}

// RegisterIPStoreDriver makes a driver available by the provided name.
//
// If this function is called twice with the same name or if the driver is nil,
// it panics.
func RegisterIPStoreDriver(name string, driver IPStoreDriver) {
	if driver == nil {
		panic("store: could not register nil ClientStoreDriver")
	}
	if _, dup := ipStoreDrivers[name]; dup {
		panic("store: could not register duplicate ClientStoreDriver: " + name)
	}
	ipStoreDrivers[name] = driver
}

// OpenIPStore returns an IPStore specified by a configuration.
func OpenIPStore(cfg *Config) (IPStore, error) {
	driver, ok := ipStoreDrivers[cfg.IPStore]
	if !ok {
		return nil, fmt.Errorf("store: unknown driver %q (forgotten import?)", cfg.IPStore)
	}

	return driver.New(cfg)
}
