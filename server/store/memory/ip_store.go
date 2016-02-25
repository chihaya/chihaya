// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"net"
	"sync"

	"github.com/mrd0ll4r/netmatch"

	"github.com/chihaya/chihaya/server/store"
)

func init() {
	store.RegisterIPStoreDriver("memory", &ipStoreDriver{})
}

type ipStoreDriver struct{}

func (d *ipStoreDriver) New(cfg *store.Config) (store.IPStore, error) {
	return &ipStore{
		ips:      make(map[[16]byte]struct{}),
		networks: netmatch.New(),
	}, nil
}

// ipStore implements store.IPStore using an in-memory map of byte arrays and
// a trie-like structure.
type ipStore struct {
	ips      map[[16]byte]struct{}
	networks *netmatch.Trie
	sync.RWMutex
}

var (
	_            store.IPStore = &ipStore{}
	v4InV6Prefix               = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}
)

// key converts an IP address to a [16]byte.
// The byte array can then be used as a key for a map, unlike net.IP, which is a
// []byte.
// If an IPv4 address is specified, it will be prefixed with
// the net.v4InV6Prefix and thus becomes a valid IPv6 address.
func key(ip net.IP) [16]byte {
	var array [16]byte

	if len(ip) == net.IPv4len {
		copy(array[:], v4InV6Prefix)
		copy(array[12:], ip)
	} else {
		copy(array[:], ip)
	}
	return array
}

func (s *ipStore) AddNetwork(network string) error {
	s.Lock()
	defer s.Unlock()

	key, length, err := netmatch.ParseNetwork(network)
	if err != nil {
		return err
	}

	return s.networks.Add(key, length)
}

func (s *ipStore) AddIP(ip net.IP) error {
	s.Lock()
	defer s.Unlock()

	s.ips[key(ip)] = struct{}{}

	return nil
}

func (s *ipStore) HasIP(ip net.IP) (bool, error) {
	s.RLock()
	defer s.RUnlock()
	key := key(ip)

	_, ok := s.ips[key]
	if ok {
		return true, nil
	}

	match, err := s.networks.Match(key)
	if err != nil {
		return false, err
	}

	return match, nil
}

func (s *ipStore) HasAnyIP(ips []net.IP) (bool, error) {
	s.RLock()
	defer s.RUnlock()

	for _, ip := range ips {
		key := key(ip)
		if _, ok := s.ips[key]; ok {
			return true, nil
		}

		match, err := s.networks.Match(key)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}

	return false, nil
}

func (s *ipStore) HasAllIPs(ips []net.IP) (bool, error) {
	s.RLock()
	defer s.RUnlock()

	for _, ip := range ips {
		key := key(ip)
		if _, ok := s.ips[key]; !ok {
			match, err := s.networks.Match(key)
			if err != nil {
				return false, err
			}
			if !match {
				return false, nil
			}
		}
	}

	return true, nil
}

func (s *ipStore) RemoveIP(ip net.IP) error {
	s.Lock()
	defer s.Unlock()

	delete(s.ips, key(ip))

	return nil
}

func (s *ipStore) RemoveNetwork(network string) error {
	s.Lock()
	defer s.Unlock()

	key, length, err := netmatch.ParseNetwork(network)
	if err != nil {
		return err
	}

	return s.networks.Remove(key, length)
}
