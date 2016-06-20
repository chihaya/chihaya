// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"testing"

	"net"

	"time"

	"github.com/chihaya/chihaya"
	"github.com/stretchr/testify/require"
)

// StringStoreTester is a collection of tests for a StringStore driver.
// Every benchmark expects a new, clean storage. Every benchmark should be
// called with a DriverConfig that ensures this.
type StringStoreTester interface {
	TestStringStore(*testing.T, *DriverConfig)
}

var _ StringStoreTester = &stringStoreTester{}

type stringStoreTester struct {
	s1, s2 string
	driver StringStoreDriver
}

// PrepareStringStoreTester prepares a reusable suite for StringStore driver
// tests.
func PrepareStringStoreTester(driver StringStoreDriver) StringStoreTester {
	return &stringStoreTester{
		s1:     "abc",
		s2:     "def",
		driver: driver,
	}
}

func (s *stringStoreTester) TestStringStore(t *testing.T, cfg *DriverConfig) {
	ss, err := s.driver.New(cfg)
	require.Nil(t, err)
	require.NotNil(t, ss)

	has, err := ss.HasString(s.s1)
	require.Nil(t, err)
	require.False(t, has)

	has, err = ss.HasString(s.s2)
	require.Nil(t, err)
	require.False(t, has)

	err = ss.RemoveString(s.s1)
	require.NotNil(t, err)

	err = ss.PutString(s.s1)
	require.Nil(t, err)

	has, err = ss.HasString(s.s1)
	require.Nil(t, err)
	require.True(t, has)

	has, err = ss.HasString(s.s2)
	require.Nil(t, err)
	require.False(t, has)

	err = ss.PutString(s.s1)
	require.Nil(t, err)

	err = ss.PutString(s.s2)
	require.Nil(t, err)

	has, err = ss.HasString(s.s1)
	require.Nil(t, err)
	require.True(t, has)

	has, err = ss.HasString(s.s2)
	require.Nil(t, err)
	require.True(t, has)

	err = ss.RemoveString(s.s1)
	require.Nil(t, err)

	err = ss.RemoveString(s.s2)
	require.Nil(t, err)

	has, err = ss.HasString(s.s1)
	require.Nil(t, err)
	require.False(t, has)

	has, err = ss.HasString(s.s2)
	require.Nil(t, err)
	require.False(t, has)

	errChan := ss.Stop()
	err = <-errChan
	require.Nil(t, err, "StringStore shutdown must not fail")
}

// IPStoreTester is a collection of tests for an IPStore driver.
// Every benchmark expects a new, clean storage. Every benchmark should be
// called with a DriverConfig that ensures this.
type IPStoreTester interface {
	TestIPStore(*testing.T, *DriverConfig)
	TestHasAllHasAny(*testing.T, *DriverConfig)
	TestNetworks(*testing.T, *DriverConfig)
	TestHasAllHasAnyNetworks(*testing.T, *DriverConfig)
}

var _ IPStoreTester = &ipStoreTester{}

type ipStoreTester struct {
	v6, v4, v4s    net.IP
	net1, net2     string
	inNet1, inNet2 net.IP
	excluded       net.IP
	driver         IPStoreDriver
}

// PrepareIPStoreTester prepares a reusable suite for IPStore driver
// tests.
func PrepareIPStoreTester(driver IPStoreDriver) IPStoreTester {
	return &ipStoreTester{
		v6:       net.ParseIP("0c22:384e:0:0c22:384e::68"),
		v4:       net.ParseIP("12.13.14.15"),
		v4s:      net.ParseIP("12.13.14.15").To4(),
		net1:     "192.168.22.255/24",
		net2:     "192.168.23.255/24",
		inNet1:   net.ParseIP("192.168.22.22"),
		inNet2:   net.ParseIP("192.168.23.23"),
		excluded: net.ParseIP("10.154.243.22"),
		driver:   driver,
	}
}

func (s *ipStoreTester) TestIPStore(t *testing.T, cfg *DriverConfig) {
	is, err := s.driver.New(cfg)
	require.Nil(t, err)
	require.NotNil(t, is)

	// check default state
	found, err := is.HasIP(s.v4)
	require.Nil(t, err)
	require.False(t, found)

	// check IPv4
	err = is.AddIP(s.v4)
	require.Nil(t, err)

	found, err = is.HasIP(s.v4)
	require.Nil(t, err)
	require.True(t, found)

	found, err = is.HasIP(s.v4s)
	require.Nil(t, err)
	require.True(t, found)

	found, err = is.HasIP(s.v6)
	require.Nil(t, err)
	require.False(t, found)

	// check removes
	err = is.RemoveIP(s.v6)
	require.NotNil(t, err)

	err = is.RemoveIP(s.v4s)
	require.Nil(t, err)

	found, err = is.HasIP(s.v4)
	require.Nil(t, err)
	require.False(t, found)

	// check IPv6
	err = is.AddIP(s.v6)
	require.Nil(t, err)

	found, err = is.HasIP(s.v6)
	require.Nil(t, err)
	require.True(t, found)

	err = is.RemoveIP(s.v6)
	require.Nil(t, err)

	found, err = is.HasIP(s.v6)
	require.Nil(t, err)
	require.False(t, found)

	errChan := is.Stop()
	err = <-errChan
	require.Nil(t, err, "IPStore shutdown must not fail")
}

func (s *ipStoreTester) TestHasAllHasAny(t *testing.T, cfg *DriverConfig) {
	is, err := s.driver.New(cfg)
	require.Nil(t, err)
	require.NotNil(t, is)

	found, err := is.HasAnyIP(nil)
	require.Nil(t, err)
	require.False(t, found)

	found, err = is.HasAllIPs(nil)
	require.Nil(t, err)
	require.True(t, found)

	found, err = is.HasAllIPs([]net.IP{s.v6})
	require.Nil(t, err)
	require.False(t, found)

	err = is.AddIP(s.v4)
	require.Nil(t, err)

	found, err = is.HasAnyIP([]net.IP{s.v6, s.v4})
	require.Nil(t, err)
	require.True(t, found)

	found, err = is.HasAllIPs([]net.IP{s.v6, s.v4})
	require.Nil(t, err)
	require.False(t, found)

	found, err = is.HasAllIPs([]net.IP{s.v4})
	require.Nil(t, err)
	require.True(t, found)

	err = is.AddIP(s.v6)
	require.Nil(t, err)

	found, err = is.HasAnyIP([]net.IP{s.v6, s.v6})
	require.Nil(t, err)
	require.True(t, found)

	found, err = is.HasAllIPs([]net.IP{s.v6, s.v6})
	require.Nil(t, err)
	require.True(t, found)

	errChan := is.Stop()
	err = <-errChan
	require.Nil(t, err, "IPStore shutdown must not fail")
}

func (s *ipStoreTester) TestNetworks(t *testing.T, cfg *DriverConfig) {
	is, err := s.driver.New(cfg)
	require.Nil(t, err)
	require.NotNil(t, is)

	match, err := is.HasIP(s.inNet1)
	require.Nil(t, err)
	require.False(t, match)

	match, err = is.HasIP(s.inNet2)
	require.Nil(t, err)
	require.False(t, match)

	err = is.AddNetwork("")
	require.NotNil(t, err)

	err = is.RemoveNetwork("")
	require.NotNil(t, err)

	err = is.AddNetwork(s.net1)
	require.Nil(t, err)

	match, err = is.HasIP(s.inNet1)
	require.Nil(t, err)
	require.True(t, match)

	match, err = is.HasIP(s.inNet2)
	require.Nil(t, err)
	require.False(t, match)

	err = is.RemoveNetwork(s.net2)
	require.NotNil(t, err)

	err = is.RemoveNetwork(s.net1)
	require.Nil(t, err)

	match, err = is.HasIP(s.inNet1)
	require.Nil(t, err)
	require.False(t, match)

	match, err = is.HasIP(s.inNet2)
	require.Nil(t, err)
	require.False(t, match)

	errChan := is.Stop()
	err = <-errChan
	require.Nil(t, err, "IPStore shutdown must not fail")
}

func (s *ipStoreTester) TestHasAllHasAnyNetworks(t *testing.T, cfg *DriverConfig) {
	is, err := s.driver.New(cfg)
	require.Nil(t, err)
	require.NotNil(t, s)

	match, err := is.HasAnyIP([]net.IP{s.inNet1, s.inNet2, s.excluded})
	require.Nil(t, err)
	require.False(t, match)

	match, err = is.HasAllIPs([]net.IP{s.inNet1, s.inNet2, s.excluded})
	require.Nil(t, err)
	require.False(t, match)

	err = is.AddNetwork(s.net1)
	require.Nil(t, err)

	match, err = is.HasAnyIP([]net.IP{s.inNet1, s.inNet2})
	require.Nil(t, err)
	require.True(t, match)

	match, err = is.HasAllIPs([]net.IP{s.inNet1, s.inNet2})
	require.Nil(t, err)
	require.False(t, match)

	err = is.AddNetwork(s.net2)
	require.Nil(t, err)

	match, err = is.HasAnyIP([]net.IP{s.inNet1, s.inNet2, s.excluded})
	require.Nil(t, err)
	require.True(t, match)

	match, err = is.HasAllIPs([]net.IP{s.inNet1, s.inNet2})
	require.Nil(t, err)
	require.True(t, match)

	match, err = is.HasAllIPs([]net.IP{s.inNet1, s.inNet2, s.excluded})
	require.Nil(t, err)
	require.False(t, match)

	err = is.RemoveNetwork(s.net1)
	require.Nil(t, err)

	match, err = is.HasAnyIP([]net.IP{s.inNet1, s.inNet2})
	require.Nil(t, err)
	require.True(t, match)

	match, err = is.HasAllIPs([]net.IP{s.inNet1, s.inNet2})
	require.Nil(t, err)
	require.False(t, match)

	err = is.RemoveNetwork(s.net2)
	require.Nil(t, err)

	match, err = is.HasAnyIP([]net.IP{s.inNet1, s.inNet2})
	require.Nil(t, err)
	require.False(t, match)

	match, err = is.HasAllIPs([]net.IP{s.inNet1, s.inNet2})
	require.Nil(t, err)
	require.False(t, match)

	errChan := is.Stop()
	err = <-errChan
	require.Nil(t, err, "IPStore shutdown must not fail")
}

// PeerStoreTester is a collection of tests for a PeerStore driver.
// Every benchmark expects a new, clean storage. Every benchmark should be
// called with a DriverConfig that ensures this.
type PeerStoreTester interface {
	TestPeerStore(*testing.T, *DriverConfig)
}

var _ PeerStoreTester = &peerStoreTester{}

type peerStoreTester struct {
	driver PeerStoreDriver
}

// PreparePeerStoreTester prepares a reusable suite for PeerStore driver
// tests.
func PreparePeerStoreTester(driver PeerStoreDriver) PeerStoreTester {
	return &peerStoreTester{
		driver: driver,
	}
}

func peerInSlice(peer chihaya.Peer, peers []chihaya.Peer) bool {
	for _, v := range peers {
		if v.Equal(peer) {
			return true
		}
	}
	return false
}

func (pt *peerStoreTester) TestPeerStore(t *testing.T, cfg *DriverConfig) {
	var (
		hash = chihaya.InfoHash([20]byte{})

		peers = []struct {
			seeder bool
			peerID string
			ip     string
			port   uint16
		}{
			{false, "-AZ3034-6wfG2wk6wWLc", "250.183.81.177", 5720},
			{false, "-AZ3042-6ozMq5q6Q3NX", "38.241.13.19", 4833},
			{false, "-BS5820-oy4La2MWGEFj", "fd45:7856:3dae::48", 2878},
			{false, "-AR6360-6oZyyMWoOOBe", "fd0a:29a8:8445::38", 3167},
			{true, "-AG2083-s1hiF8vGAAg0", "231.231.49.173", 1453},
			{true, "-AG3003-lEl2Mm4NEO4n", "254.99.84.77", 7032},
			{true, "-MR1100-00HS~T7*65rm", "211.229.100.129", 2614},
			{true, "-LK0140-ATIV~nbEQAMr", "fdad:c435:bf79::12", 4114},
			{true, "-KT2210-347143496631", "fdda:1b35:7d6e::9", 6179},
			{true, "-TR0960-6ep6svaa61r4", "fd7f:78f0:4c77::55", 4727},
		}
	)
	s, err := pt.driver.New(cfg)
	require.Nil(t, err)
	require.NotNil(t, s)

	for _, p := range peers {
		// Construct chihaya.Peer from test data.
		peer := chihaya.Peer{
			ID:   chihaya.PeerIDFromString(p.peerID),
			IP:   net.ParseIP(p.ip),
			Port: p.port,
		}

		if p.seeder {
			err = s.PutSeeder(hash, peer)
		} else {
			err = s.PutLeecher(hash, peer)
		}
		require.Nil(t, err)
	}

	leechers1, leechers61, err := s.GetLeechers(hash)
	require.Nil(t, err)
	require.NotEmpty(t, leechers1)
	require.NotEmpty(t, leechers61)
	num := s.NumLeechers(hash)
	require.Equal(t, len(leechers1)+len(leechers61), num)

	seeders1, seeders61, err := s.GetSeeders(hash)
	require.Nil(t, err)
	require.NotEmpty(t, seeders1)
	require.NotEmpty(t, seeders61)
	num = s.NumSeeders(hash)
	require.Equal(t, len(seeders1)+len(seeders61), num)

	leechers := append(leechers1, leechers61...)
	seeders := append(seeders1, seeders61...)

	for _, p := range peers {
		// Construct chihaya.Peer from test data.
		peer := chihaya.Peer{
			ID:   chihaya.PeerIDFromString(p.peerID),
			IP:   net.ParseIP(p.ip),
			Port: p.port,
		}

		if p.seeder {
			require.True(t, peerInSlice(peer, seeders))
		} else {
			require.True(t, peerInSlice(peer, leechers))
		}

		if p.seeder {
			err = s.DeleteSeeder(hash, peer)
		} else {
			err = s.DeleteLeecher(hash, peer)
		}
		require.Nil(t, err)
	}

	require.Zero(t, s.NumLeechers(hash))
	require.Zero(t, s.NumSeeders(hash))

	// Re-add all the peers to the peerStore.
	for _, p := range peers {
		// Construct chihaya.Peer from test data.
		peer := chihaya.Peer{
			ID:   chihaya.PeerIDFromString(p.peerID),
			IP:   net.ParseIP(p.ip),
			Port: p.port,
		}
		if p.seeder {
			s.PutSeeder(hash, peer)
		} else {
			s.PutLeecher(hash, peer)
		}
	}

	// Check that there are 6 seeders, and 4 leechers.
	require.Equal(t, 6, s.NumSeeders(hash))
	require.Equal(t, 4, s.NumLeechers(hash))
	peer := chihaya.Peer{
		ID:   chihaya.PeerIDFromString(peers[0].peerID),
		IP:   net.ParseIP(peers[0].ip),
		Port: peers[0].port,
	}
	err = s.GraduateLeecher(hash, peer)
	require.Nil(t, err)
	// Check that there are 7 seeders, and 3 leechers after graduating a
	// leecher to a seeder.
	require.Equal(t, 7, s.NumSeeders(hash))
	require.Equal(t, 3, s.NumLeechers(hash))

	peers1, peers61, err := s.AnnouncePeers(hash, true, 5, peer, chihaya.Peer{})
	require.Nil(t, err)
	require.NotNil(t, peers1)
	require.NotNil(t, peers61)

	err = s.CollectGarbage(time.Now())
	require.Nil(t, err)
	require.Equal(t, s.NumLeechers(hash), 0)
	require.Equal(t, s.NumSeeders(hash), 0)

	errChan := s.Stop()
	err = <-errChan
	require.Nil(t, err, "PeerStore shutdown must not fail")
}
