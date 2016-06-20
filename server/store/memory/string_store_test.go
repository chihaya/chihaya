// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"testing"

	"github.com/chihaya/chihaya/server/store"
)

var (
	driver                 = &stringStoreDriver{}
	stringStoreTester      = store.PrepareStringStoreTester(driver)
	stringStoreBenchmarker = store.PrepareStringStoreBenchmarker(&stringStoreDriver{})
)

func TestStringStore(t *testing.T) {
	stringStoreTester.TestStringStore(t, &store.DriverConfig{})
}

func BenchmarkStringStore_AddShort(b *testing.B) {
	stringStoreBenchmarker.AddShort(b, &store.DriverConfig{})
}

func BenchmarkStringStore_AddLong(b *testing.B) {
	stringStoreBenchmarker.AddLong(b, &store.DriverConfig{})
}

func BenchmarkStringStore_LookupShort(b *testing.B) {
	stringStoreBenchmarker.LookupShort(b, &store.DriverConfig{})
}

func BenchmarkStringStore_LookupLong(b *testing.B) {
	stringStoreBenchmarker.LookupLong(b, &store.DriverConfig{})
}

func BenchmarkStringStore_AddRemoveShort(b *testing.B) {
	stringStoreBenchmarker.AddRemoveShort(b, &store.DriverConfig{})
}

func BenchmarkStringStore_AddRemoveLong(b *testing.B) {
	stringStoreBenchmarker.AddRemoveLong(b, &store.DriverConfig{})
}

func BenchmarkStringStore_LookupNonExistShort(b *testing.B) {
	stringStoreBenchmarker.LookupNonExistShort(b, &store.DriverConfig{})
}

func BenchmarkStringStore_LookupNonExistLong(b *testing.B) {
	stringStoreBenchmarker.LookupNonExistLong(b, &store.DriverConfig{})
}

func BenchmarkStringStore_RemoveNonExistShort(b *testing.B) {
	stringStoreBenchmarker.RemoveNonExistShort(b, &store.DriverConfig{})
}

func BenchmarkStringStore_RemoveNonExistLong(b *testing.B) {
	stringStoreBenchmarker.RemoveNonExistLong(b, &store.DriverConfig{})
}

func BenchmarkStringStore_Add1KShort(b *testing.B) {
	stringStoreBenchmarker.Add1KShort(b, &store.DriverConfig{})
}

func BenchmarkStringStore_Add1KLong(b *testing.B) {
	stringStoreBenchmarker.Add1KLong(b, &store.DriverConfig{})
}

func BenchmarkStringStore_Lookup1KShort(b *testing.B) {
	stringStoreBenchmarker.Lookup1KShort(b, &store.DriverConfig{})
}

func BenchmarkStringStore_Lookup1KLong(b *testing.B) {
	stringStoreBenchmarker.Lookup1KLong(b, &store.DriverConfig{})
}

func BenchmarkStringStore_AddRemove1KShort(b *testing.B) {
	stringStoreBenchmarker.AddRemove1KShort(b, &store.DriverConfig{})
}

func BenchmarkStringStore_AddRemove1KLong(b *testing.B) {
	stringStoreBenchmarker.AddRemove1KLong(b, &store.DriverConfig{})
}

func BenchmarkStringStore_LookupNonExist1KShort(b *testing.B) {
	stringStoreBenchmarker.LookupNonExist1KShort(b, &store.DriverConfig{})
}

func BenchmarkStringStore_LookupNonExist1KLong(b *testing.B) {
	stringStoreBenchmarker.LookupNonExist1KLong(b, &store.DriverConfig{})
}

func BenchmarkStringStore_RemoveNonExist1KShort(b *testing.B) {
	stringStoreBenchmarker.RemoveNonExist1KShort(b, &store.DriverConfig{})
}

func BenchmarkStringStore_RemoveNonExist1KLong(b *testing.B) {
	stringStoreBenchmarker.RemoveNonExist1KLong(b, &store.DriverConfig{})
}
