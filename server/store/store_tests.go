// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"testing"

	"net"

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
