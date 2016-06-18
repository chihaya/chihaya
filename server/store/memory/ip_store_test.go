// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"net"
	"testing"

	"github.com/chihaya/chihaya/server/store"

	"github.com/stretchr/testify/require"
)

var (
	v6  = net.ParseIP("0c22:384e:0:0c22:384e::68")
	v4  = net.ParseIP("12.13.14.15")
	v4s = net.ParseIP("12.13.14.15").To4()

	ipStoreTester = store.PrepareIPStoreTester(&ipStoreDriver{})
)

func TestKey(t *testing.T) {
	var table = []struct {
		input    net.IP
		expected [16]byte
	}{
		{v6, [16]byte{12, 34, 56, 78, 0, 0, 12, 34, 56, 78, 0, 0, 0, 0, 0, 104}},
		{v4, [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 12, 13, 14, 15}},  // IPv4 in IPv6 prefix
		{v4s, [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 12, 13, 14, 15}}, // is equal to the one above, should produce equal output
	}

	for _, tt := range table {
		got := key(tt.input)
		require.Equal(t, got, tt.expected)
	}
}

func TestIPStore(t *testing.T) {
	ipStoreTester.TestIPStore(t, &store.DriverConfig{})
}

func TestHasAllHasAny(t *testing.T) {
	ipStoreTester.TestHasAllHasAny(t, &store.DriverConfig{})
}

func TestNetworks(t *testing.T) {
	ipStoreTester.TestNetworks(t, &store.DriverConfig{})
}

func TestHasAllHasAnyNetworks(t *testing.T) {
	ipStoreTester.TestHasAllHasAnyNetworks(t, &store.DriverConfig{})
}
