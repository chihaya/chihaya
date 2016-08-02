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

	ipStoreTester      = store.PrepareIPStoreTester(&ipStoreDriver{})
	ipStoreBenchmarker = store.PrepareIPStoreBenchmarker(&ipStoreDriver{})
	ipStoreTestConfig  = &store.DriverConfig{}
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
	ipStoreTester.TestIPStore(t, ipStoreTestConfig)
}

func TestHasAllHasAny(t *testing.T) {
	ipStoreTester.TestHasAllHasAny(t, ipStoreTestConfig)
}

func TestNetworks(t *testing.T) {
	ipStoreTester.TestNetworks(t, ipStoreTestConfig)
}

func TestHasAllHasAnyNetworks(t *testing.T) {
	ipStoreTester.TestHasAllHasAnyNetworks(t, ipStoreTestConfig)
}

func BenchmarkIPStore_AddV4(b *testing.B) {
	ipStoreBenchmarker.AddV4(b, ipStoreTestConfig)
}

func BenchmarkIPStore_AddV6(b *testing.B) {
	ipStoreBenchmarker.AddV6(b, ipStoreTestConfig)
}

func BenchmarkIPStore_LookupV4(b *testing.B) {
	ipStoreBenchmarker.LookupV4(b, ipStoreTestConfig)
}

func BenchmarkIPStore_LookupV6(b *testing.B) {
	ipStoreBenchmarker.LookupV6(b, ipStoreTestConfig)
}

func BenchmarkIPStore_AddRemoveV4(b *testing.B) {
	ipStoreBenchmarker.AddRemoveV4(b, ipStoreTestConfig)
}

func BenchmarkIPStore_AddRemoveV6(b *testing.B) {
	ipStoreBenchmarker.AddRemoveV6(b, ipStoreTestConfig)
}

func BenchmarkIPStore_LookupNonExistV4(b *testing.B) {
	ipStoreBenchmarker.LookupNonExistV4(b, ipStoreTestConfig)
}

func BenchmarkIPStore_LookupNonExistV6(b *testing.B) {
	ipStoreBenchmarker.LookupNonExistV6(b, ipStoreTestConfig)
}

func BenchmarkIPStore_RemoveNonExistV4(b *testing.B) {
	ipStoreBenchmarker.RemoveNonExistV4(b, ipStoreTestConfig)
}

func BenchmarkIPStore_RemoveNonExistV6(b *testing.B) {
	ipStoreBenchmarker.RemoveNonExistV6(b, ipStoreTestConfig)
}

func BenchmarkIPStore_AddV4Network(b *testing.B) {
	ipStoreBenchmarker.AddV4Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_AddV6Network(b *testing.B) {
	ipStoreBenchmarker.AddV6Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_LookupV4Network(b *testing.B) {
	ipStoreBenchmarker.LookupV4Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_LookupV6Network(b *testing.B) {
	ipStoreBenchmarker.LookupV6Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_AddRemoveV4Network(b *testing.B) {
	ipStoreBenchmarker.AddRemoveV4Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_AddRemoveV6Network(b *testing.B) {
	ipStoreBenchmarker.AddRemoveV6Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_RemoveNonExistV4Network(b *testing.B) {
	ipStoreBenchmarker.RemoveNonExistV4Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_RemoveNonExistV6Network(b *testing.B) {
	ipStoreBenchmarker.RemoveNonExistV6Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_Add1KV4(b *testing.B) {
	ipStoreBenchmarker.Add1KV4(b, ipStoreTestConfig)
}

func BenchmarkIPStore_Add1KV6(b *testing.B) {
	ipStoreBenchmarker.Add1KV6(b, ipStoreTestConfig)
}

func BenchmarkIPStore_Lookup1KV4(b *testing.B) {
	ipStoreBenchmarker.Lookup1KV4(b, ipStoreTestConfig)
}

func BenchmarkIPStore_Lookup1KV6(b *testing.B) {
	ipStoreBenchmarker.Lookup1KV6(b, ipStoreTestConfig)
}

func BenchmarkIPStore_AddRemove1KV4(b *testing.B) {
	ipStoreBenchmarker.AddRemove1KV4(b, ipStoreTestConfig)
}

func BenchmarkIPStore_AddRemove1KV6(b *testing.B) {
	ipStoreBenchmarker.AddRemove1KV6(b, ipStoreTestConfig)
}

func BenchmarkIPStore_LookupNonExist1KV4(b *testing.B) {
	ipStoreBenchmarker.LookupNonExist1KV4(b, ipStoreTestConfig)
}

func BenchmarkIPStore_LookupNonExist1KV6(b *testing.B) {
	ipStoreBenchmarker.LookupNonExist1KV6(b, ipStoreTestConfig)
}

func BenchmarkIPStore_RemoveNonExist1KV4(b *testing.B) {
	ipStoreBenchmarker.RemoveNonExist1KV4(b, ipStoreTestConfig)
}

func BenchmarkIPStore_RemoveNonExist1KV6(b *testing.B) {
	ipStoreBenchmarker.RemoveNonExist1KV6(b, ipStoreTestConfig)
}

func BenchmarkIPStore_Add1KV4Network(b *testing.B) {
	ipStoreBenchmarker.Add1KV4Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_Add1KV6Network(b *testing.B) {
	ipStoreBenchmarker.Add1KV6Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_Lookup1KV4Network(b *testing.B) {
	ipStoreBenchmarker.Lookup1KV4Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_Lookup1KV6Network(b *testing.B) {
	ipStoreBenchmarker.Lookup1KV6Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_AddRemove1KV4Network(b *testing.B) {
	ipStoreBenchmarker.AddRemove1KV4Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_AddRemove1KV6Network(b *testing.B) {
	ipStoreBenchmarker.AddRemove1KV6Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_RemoveNonExist1KV4Network(b *testing.B) {
	ipStoreBenchmarker.RemoveNonExist1KV4Network(b, ipStoreTestConfig)
}

func BenchmarkIPStore_RemoveNonExist1KV6Network(b *testing.B) {
	ipStoreBenchmarker.RemoveNonExist1KV6Network(b, ipStoreTestConfig)
}
