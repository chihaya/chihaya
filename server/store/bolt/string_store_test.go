// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package bolt

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chihaya/chihaya/server/store"
)

var (
	driver                 = &stringStoreDriver{}
	stringStoreTester      = store.PrepareStringStoreTester(driver)
	stringStoreBenchmarker = store.PrepareStringStoreBenchmarker(driver)
)

const benchDB = "bench.db"

func TestStringStoreDriver_New(t *testing.T) {
	ss, err := driver.New(nil)
	require.Equal(t, ErrMissingConfig, err)
	require.Nil(t, ss)

	f, err := os.Create("test.db")
	require.Nil(t, err)
	require.NotNil(t, f)
	defer os.Remove("test.db")
	defer f.Close()

	fmt.Fprint(f, "garbage")

	ss, err = driver.New(&store.DriverConfig{Config: boltConfig{File: "test.db"}})
	require.NotNil(t, err)
	require.Nil(t, ss)
}

func TestStringStore(t *testing.T) {
	defer os.Remove("test.db")
	stringStoreTester.TestStringStore(t, &store.DriverConfig{Config: boltConfig{File: "test.db"}})
}

type benchmarkFunc func(*testing.B, *store.DriverConfig)

func benchmarkHarness(f benchmarkFunc, b *testing.B) {
	defer os.Remove(benchDB)
	f(b, &store.DriverConfig{Config: boltConfig{File: benchDB}})
}

func BenchmarkStringStore_AddShort(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.AddShort, b)
}

func BenchmarkStringStore_AddLong(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.AddLong, b)
}

func BenchmarkStringStore_LookupShort(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.LookupShort, b)
}

func BenchmarkStringStore_LookupLong(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.LookupLong, b)
}

func BenchmarkStringStore_AddRemoveShort(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.AddRemoveShort, b)
}

func BenchmarkStringStore_AddRemoveLong(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.AddRemoveLong, b)
}

func BenchmarkStringStore_LookupNonExistShort(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.LookupNonExistShort, b)
}

func BenchmarkStringStore_LookupNonExistLong(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.LookupNonExistLong, b)
}

func BenchmarkStringStore_RemoveNonExistShort(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.RemoveNonExistShort, b)
}

func BenchmarkStringStore_RemoveNonExistLong(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.RemoveNonExistLong, b)
}

func BenchmarkStringStore_Add1KShort(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.Add1KShort, b)
}

func BenchmarkStringStore_Add1KLong(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.Add1KLong, b)
}

func BenchmarkStringStore_Lookup1KShort(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.Lookup1KShort, b)
}

func BenchmarkStringStore_Lookup1KLong(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.Lookup1KLong, b)
}

func BenchmarkStringStore_AddRemove1KShort(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.AddRemove1KShort, b)
}

func BenchmarkStringStore_AddRemove1KLong(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.AddRemove1KLong, b)
}

func BenchmarkStringStore_LookupNonExist1KShort(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.LookupNonExist1KShort, b)
}

func BenchmarkStringStore_LookupNonExist1KLong(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.LookupNonExist1KLong, b)
}

func BenchmarkStringStore_RemoveNonExist1KShort(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.RemoveNonExist1KShort, b)
}

func BenchmarkStringStore_RemoveNonExist1KLong(b *testing.B) {
	benchmarkHarness(stringStoreBenchmarker.RemoveNonExist1KLong, b)
}
