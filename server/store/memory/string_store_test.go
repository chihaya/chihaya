// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"testing"

	"github.com/chihaya/chihaya/server/store"
)

var (
	stringStoreTester      = store.PrepareStringStoreTester(&stringStoreDriver{})
	stringStoreBenchmarker = store.PrepareStringStoreBenchmarker(&stringStoreDriver{})
	stringStoreTestConfig  = &store.DriverConfig{}
)

func TestStringStore(t *testing.T) {
	stringStoreTester.TestStringStore(t, stringStoreTestConfig)
}

func BenchmarkStringStore_AddShort(b *testing.B) {
	stringStoreBenchmarker.AddShort(b, stringStoreTestConfig)
}

func BenchmarkStringStore_AddLong(b *testing.B) {
	stringStoreBenchmarker.AddLong(b, stringStoreTestConfig)
}

func BenchmarkStringStore_LookupShort(b *testing.B) {
	stringStoreBenchmarker.LookupShort(b, stringStoreTestConfig)
}

func BenchmarkStringStore_LookupLong(b *testing.B) {
	stringStoreBenchmarker.LookupLong(b, stringStoreTestConfig)
}

func BenchmarkStringStore_AddRemoveShort(b *testing.B) {
	stringStoreBenchmarker.AddRemoveShort(b, stringStoreTestConfig)
}

func BenchmarkStringStore_AddRemoveLong(b *testing.B) {
	stringStoreBenchmarker.AddRemoveLong(b, stringStoreTestConfig)
}

func BenchmarkStringStore_LookupNonExistShort(b *testing.B) {
	stringStoreBenchmarker.LookupNonExistShort(b, stringStoreTestConfig)
}

func BenchmarkStringStore_LookupNonExistLong(b *testing.B) {
	stringStoreBenchmarker.LookupNonExistLong(b, stringStoreTestConfig)
}

func BenchmarkStringStore_RemoveNonExistShort(b *testing.B) {
	stringStoreBenchmarker.RemoveNonExistShort(b, stringStoreTestConfig)
}

func BenchmarkStringStore_RemoveNonExistLong(b *testing.B) {
	stringStoreBenchmarker.RemoveNonExistLong(b, stringStoreTestConfig)
}

func BenchmarkStringStore_Add1KShort(b *testing.B) {
	stringStoreBenchmarker.Add1KShort(b, stringStoreTestConfig)
}

func BenchmarkStringStore_Add1KLong(b *testing.B) {
	stringStoreBenchmarker.Add1KLong(b, stringStoreTestConfig)
}

func BenchmarkStringStore_Lookup1KShort(b *testing.B) {
	stringStoreBenchmarker.Lookup1KShort(b, stringStoreTestConfig)
}

func BenchmarkStringStore_Lookup1KLong(b *testing.B) {
	stringStoreBenchmarker.Lookup1KLong(b, stringStoreTestConfig)
}

func BenchmarkStringStore_AddRemove1KShort(b *testing.B) {
	stringStoreBenchmarker.AddRemove1KShort(b, stringStoreTestConfig)
}

func BenchmarkStringStore_AddRemove1KLong(b *testing.B) {
	stringStoreBenchmarker.AddRemove1KLong(b, stringStoreTestConfig)
}

func BenchmarkStringStore_LookupNonExist1KShort(b *testing.B) {
	stringStoreBenchmarker.LookupNonExist1KShort(b, stringStoreTestConfig)
}

func BenchmarkStringStore_LookupNonExist1KLong(b *testing.B) {
	stringStoreBenchmarker.LookupNonExist1KLong(b, stringStoreTestConfig)
}

func BenchmarkStringStore_RemoveNonExist1KShort(b *testing.B) {
	stringStoreBenchmarker.RemoveNonExist1KShort(b, stringStoreTestConfig)
}

func BenchmarkStringStore_RemoveNonExist1KLong(b *testing.B) {
	stringStoreBenchmarker.RemoveNonExist1KLong(b, stringStoreTestConfig)
}
