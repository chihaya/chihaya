// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const num1KElements = 1000

// StringStoreBenchmarker is a collection of benchmarks for StringStore drivers.
// Every benchmark expects a new, clean storage. Every benchmark should be
// called with a DriverConfig that ensures this.
type StringStoreBenchmarker interface {
	AddShort(*testing.B, *DriverConfig)
	AddLong(*testing.B, *DriverConfig)
	LookupShort(*testing.B, *DriverConfig)
	LookupLong(*testing.B, *DriverConfig)
	AddRemoveShort(*testing.B, *DriverConfig)
	AddRemoveLong(*testing.B, *DriverConfig)
	LookupNonExistShort(*testing.B, *DriverConfig)
	LookupNonExistLong(*testing.B, *DriverConfig)
	RemoveNonExistShort(*testing.B, *DriverConfig)
	RemoveNonExistLong(*testing.B, *DriverConfig)

	Add1KShort(*testing.B, *DriverConfig)
	Add1KLong(*testing.B, *DriverConfig)
	Lookup1KShort(*testing.B, *DriverConfig)
	Lookup1KLong(*testing.B, *DriverConfig)
	AddRemove1KShort(*testing.B, *DriverConfig)
	AddRemove1KLong(*testing.B, *DriverConfig)
	LookupNonExist1KShort(*testing.B, *DriverConfig)
	LookupNonExist1KLong(*testing.B, *DriverConfig)
	RemoveNonExist1KShort(*testing.B, *DriverConfig)
	RemoveNonExist1KLong(*testing.B, *DriverConfig)
}

var _ StringStoreBenchmarker = &stringStoreBench{}

type stringStoreBench struct {
	// sShort holds differentStrings unique strings of length 10.
	sShort [num1KElements]string
	// sLong holds differentStrings unique strings of length 1000.
	sLong [num1KElements]string

	driver StringStoreDriver
}

func generateLongStrings() (a [num1KElements]string) {
	b := make([]byte, 2)
	for i := range a {
		b[0] = byte(i)
		b[1] = byte(i >> 8)
		a[i] = strings.Repeat(fmt.Sprintf("%x", b), 250)
	}

	return
}

func generateShortStrings() (a [num1KElements]string) {
	b := make([]byte, 2)
	for i := range a {
		b[0] = byte(i)
		b[1] = byte(i >> 8)
		a[i] = strings.Repeat(fmt.Sprintf("%x", b), 3)[:10]
	}

	return
}

// PrepareStringStoreBenchmarker prepares a reusable suite for StringStore driver
// benchmarks.
func PrepareStringStoreBenchmarker(driver StringStoreDriver) StringStoreBenchmarker {
	return stringStoreBench{
		sShort: generateShortStrings(),
		sLong:  generateLongStrings(),
		driver: driver,
	}
}

type stringStoreSetupFunc func(StringStore) error

func stringStoreSetupNOP(StringStore) error { return nil }

type stringStoreBenchFunc func(StringStore, int) error

func (sb stringStoreBench) runBenchmark(b *testing.B, cfg *DriverConfig, setup stringStoreSetupFunc, execute stringStoreBenchFunc) {
	ss, err := sb.driver.New(cfg)
	require.Nil(b, err, "Constructor error must be nil")
	require.NotNil(b, ss, "String store must not be nil")

	err = setup(ss)
	require.Nil(b, err, "Benchmark setup must not fail")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execute(ss, i)
	}
	b.StopTimer()

	errChan := ss.Stop()
	err = <-errChan
	require.Nil(b, err, "StringStore shutdown must not fail")
}

func (sb stringStoreBench) AddShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sShort[0])
			return nil
		})
}

func (sb stringStoreBench) AddLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sLong[0])
			return nil
		})
}

func (sb stringStoreBench) Add1KShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sShort[i%num1KElements])
			return nil
		})
}

func (sb stringStoreBench) Add1KLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sLong[i%num1KElements])
			return nil
		})
}

func (sb stringStoreBench) LookupShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg,
		func(ss StringStore) error {
			return ss.PutString(sb.sShort[0])
		},
		func(ss StringStore, i int) error {
			ss.HasString(sb.sShort[0])
			return nil
		})
}

func (sb stringStoreBench) LookupLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg,
		func(ss StringStore) error {
			return ss.PutString(sb.sLong[0])
		},
		func(ss StringStore, i int) error {
			ss.HasString(sb.sLong[0])
			return nil
		})
}

func (sb stringStoreBench) Lookup1KShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg,
		func(ss StringStore) error {
			for i := 0; i < num1KElements; i++ {
				err := ss.PutString(sb.sShort[i])
				if err != nil {
					return err
				}
			}
			return nil
		},
		func(ss StringStore, i int) error {
			ss.HasString(sb.sShort[i%num1KElements])
			return nil
		})
}

func (sb stringStoreBench) Lookup1KLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg,
		func(ss StringStore) error {
			for i := 0; i < num1KElements; i++ {
				err := ss.PutString(sb.sLong[i])
				if err != nil {
					return err
				}
			}
			return nil
		},
		func(ss StringStore, i int) error {
			ss.HasString(sb.sLong[i%num1KElements])
			return nil
		})
}

func (sb stringStoreBench) AddRemoveShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sShort[0])
			ss.RemoveString(sb.sShort[0])
			return nil
		})
}

func (sb stringStoreBench) AddRemoveLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sLong[0])
			ss.RemoveString(sb.sLong[0])
			return nil
		})
}

func (sb stringStoreBench) AddRemove1KShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sShort[i%num1KElements])
			ss.RemoveString(sb.sShort[i%num1KElements])
			return nil
		})
}

func (sb stringStoreBench) AddRemove1KLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sLong[i%num1KElements])
			ss.RemoveString(sb.sLong[i%num1KElements])
			return nil
		})
}

func (sb stringStoreBench) LookupNonExistShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sShort[0])
			return nil
		})
}

func (sb stringStoreBench) LookupNonExistLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sLong[0])
			return nil
		})
}

func (sb stringStoreBench) LookupNonExist1KShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sShort[i%num1KElements])
			return nil
		})
}

func (sb stringStoreBench) LookupNonExist1KLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sLong[i%num1KElements])
			return nil
		})
}

func (sb stringStoreBench) RemoveNonExistShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.RemoveString(sb.sShort[0])
			return nil
		})
}

func (sb stringStoreBench) RemoveNonExistLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.RemoveString(sb.sLong[0])
			return nil
		})
}

func (sb stringStoreBench) RemoveNonExist1KShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.RemoveString(sb.sShort[i%num1KElements])
			return nil
		})
}

func (sb stringStoreBench) RemoveNonExist1KLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.RemoveString(sb.sLong[i%num1KElements])
			return nil
		})
}
