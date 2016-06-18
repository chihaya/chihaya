// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"fmt"
	"net"
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

// IPStoreBenchmarker is a collection of benchmarks for IPStore drivers.
// Every benchmark expects a new, clean storage. Every benchmark should be
// called with a DriverConfig that ensures this.
type IPStoreBenchmarker interface {
	AddV4(*testing.B, *DriverConfig)
	AddV6(*testing.B, *DriverConfig)
	LookupV4(*testing.B, *DriverConfig)
	LookupV6(*testing.B, *DriverConfig)
	AddRemoveV4(*testing.B, *DriverConfig)
	AddRemoveV6(*testing.B, *DriverConfig)
	LookupNonExistV4(*testing.B, *DriverConfig)
	LookupNonExistV6(*testing.B, *DriverConfig)
	RemoveNonExistV4(*testing.B, *DriverConfig)
	RemoveNonExistV6(*testing.B, *DriverConfig)

	AddV4Network(*testing.B, *DriverConfig)
	AddV6Network(*testing.B, *DriverConfig)
	LookupV4Network(*testing.B, *DriverConfig)
	LookupV6Network(*testing.B, *DriverConfig)
	AddRemoveV4Network(*testing.B, *DriverConfig)
	AddRemoveV6Network(*testing.B, *DriverConfig)
	RemoveNonExistV4Network(*testing.B, *DriverConfig)
	RemoveNonExistV6Network(*testing.B, *DriverConfig)

	Add1KV4(*testing.B, *DriverConfig)
	Add1KV6(*testing.B, *DriverConfig)
	Lookup1KV4(*testing.B, *DriverConfig)
	Lookup1KV6(*testing.B, *DriverConfig)
	AddRemove1KV4(*testing.B, *DriverConfig)
	AddRemove1KV6(*testing.B, *DriverConfig)
	LookupNonExist1KV4(*testing.B, *DriverConfig)
	LookupNonExist1KV6(*testing.B, *DriverConfig)
	RemoveNonExist1KV4(*testing.B, *DriverConfig)
	RemoveNonExist1KV6(*testing.B, *DriverConfig)

	Add1KV4Network(*testing.B, *DriverConfig)
	Add1KV6Network(*testing.B, *DriverConfig)
	Lookup1KV4Network(*testing.B, *DriverConfig)
	Lookup1KV6Network(*testing.B, *DriverConfig)
	AddRemove1KV4Network(*testing.B, *DriverConfig)
	AddRemove1KV6Network(*testing.B, *DriverConfig)
	RemoveNonExist1KV4Network(*testing.B, *DriverConfig)
	RemoveNonExist1KV6Network(*testing.B, *DriverConfig)
}

func generateV4Networks() (a [num1KElements]string) {
	b := make([]byte, 2)
	for i := range a {
		b[0] = byte(i)
		b[1] = byte(i >> 8)
		a[i] = fmt.Sprintf("64.%d.%d.255/24", b[0], b[1])
	}

	return
}

func generateV6Networks() (a [num1KElements]string) {
	b := make([]byte, 2)
	for i := range a {
		b[0] = byte(i)
		b[1] = byte(i >> 8)
		a[i] = fmt.Sprintf("6464:6464:6464:%02x%02x:ffff:ffff:ffff:ffff/64", b[0], b[1])
	}

	return
}

func generateV4IPs() (a [num1KElements]net.IP) {
	b := make([]byte, 2)
	for i := range a {
		b[0] = byte(i)
		b[1] = byte(i >> 8)
		a[i] = net.ParseIP(fmt.Sprintf("64.%d.%d.64", b[0], b[1])).To4()
	}

	return
}

func generateV6IPs() (a [num1KElements]net.IP) {
	b := make([]byte, 2)
	for i := range a {
		b[0] = byte(i)
		b[1] = byte(i >> 8)
		a[i] = net.ParseIP(fmt.Sprintf("6464:6464:6464:%02x%02x:6464:6464:6464:6464", b[0], b[1]))
	}

	return
}

type ipStoreBench struct {
	v4IPs [num1KElements]net.IP
	v6IPs [num1KElements]net.IP

	v4Networks [num1KElements]string
	v6Networks [num1KElements]string

	driver IPStoreDriver
}

// PrepareIPStoreBenchmarker prepares a reusable suite for StringStore driver
// benchmarks.
func PrepareIPStoreBenchmarker(driver IPStoreDriver) IPStoreBenchmarker {
	return ipStoreBench{
		v4IPs:      generateV4IPs(),
		v6IPs:      generateV6IPs(),
		v4Networks: generateV4Networks(),
		v6Networks: generateV6Networks(),
		driver:     driver,
	}
}

type ipStoreSetupFunc func(IPStore) error

func ipStoreSetupNOP(IPStore) error { return nil }

type ipStoreBenchFunc func(IPStore, int) error

func (ib ipStoreBench) runBenchmark(b *testing.B, cfg *DriverConfig, setup ipStoreSetupFunc, execute ipStoreBenchFunc) {
	is, err := ib.driver.New(cfg)
	require.Nil(b, err, "Constructor error must be nil")
	require.NotNil(b, is, "IP store must not be nil")

	err = setup(is)
	require.Nil(b, err, "Benchmark setup must not fail")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execute(is, i)
	}
	b.StopTimer()

	errChan := is.Stop()
	err = <-errChan
	require.Nil(b, err, "IPStore shutdown must not fail")
}

func (ib ipStoreBench) AddV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v4IPs[0])
			return nil
		})
}

func (ib ipStoreBench) AddV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v6IPs[0])
			return nil
		})
}

func (ib ipStoreBench) LookupV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg,
		func(is IPStore) error {
			return is.AddIP(ib.v4IPs[0])
		},
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[0])
			return nil
		})
}

func (ib ipStoreBench) LookupV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg,
		func(is IPStore) error {
			return is.AddIP(ib.v6IPs[0])
		},
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[0])
			return nil
		})
}

func (ib ipStoreBench) AddRemoveV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v4IPs[0])
			is.RemoveIP(ib.v4IPs[0])
			return nil
		})
}

func (ib ipStoreBench) AddRemoveV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v6IPs[0])
			is.RemoveIP(ib.v6IPs[0])
			return nil
		})
}

func (ib ipStoreBench) LookupNonExistV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[0])
			return nil
		})
}

func (ib ipStoreBench) LookupNonExistV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[0])
			return nil
		})
}

func (ib ipStoreBench) RemoveNonExistV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveIP(ib.v4IPs[0])
			return nil
		})
}

func (ib ipStoreBench) RemoveNonExistV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveIP(ib.v6IPs[0])
			return nil
		})
}

func (ib ipStoreBench) AddV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v4Networks[0])
			return nil
		})
}

func (ib ipStoreBench) AddV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v6Networks[0])
			return nil
		})
}

func (ib ipStoreBench) LookupV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg,
		func(is IPStore) error {
			return is.AddNetwork(ib.v4Networks[0])
		},
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[0])
			return nil
		})
}

func (ib ipStoreBench) LookupV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg,
		func(is IPStore) error {
			return is.AddNetwork(ib.v6Networks[0])
		},
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[0])
			return nil
		})
}

func (ib ipStoreBench) AddRemoveV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v4Networks[0])
			is.RemoveNetwork(ib.v4Networks[0])
			return nil
		})
}

func (ib ipStoreBench) AddRemoveV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v6Networks[0])
			is.RemoveNetwork(ib.v6Networks[0])
			return nil
		})
}

func (ib ipStoreBench) RemoveNonExistV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveNetwork(ib.v4Networks[0])
			return nil
		})
}

func (ib ipStoreBench) RemoveNonExistV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveNetwork(ib.v6Networks[0])
			return nil
		})
}

func (ib ipStoreBench) Add1KV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) Add1KV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) Lookup1KV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg,
		func(is IPStore) error {
			for i := 0; i < num1KElements; i++ {
				err := is.AddIP(ib.v4IPs[i%num1KElements])
				if err != nil {
					return err
				}
			}
			return nil
		},
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) Lookup1KV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg,
		func(is IPStore) error {
			for i := 0; i < num1KElements; i++ {
				err := is.AddIP(ib.v6IPs[i%num1KElements])
				if err != nil {
					return err
				}
			}
			return nil
		},
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) AddRemove1KV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v4IPs[i%num1KElements])
			is.RemoveIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) AddRemove1KV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v6IPs[i%num1KElements])
			is.RemoveIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) LookupNonExist1KV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) LookupNonExist1KV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) RemoveNonExist1KV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) RemoveNonExist1KV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) Add1KV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v4Networks[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) Add1KV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v6Networks[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) Lookup1KV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg,
		func(is IPStore) error {
			for i := 0; i < num1KElements; i++ {
				err := is.AddNetwork(ib.v4Networks[i%num1KElements])
				if err != nil {
					return err
				}
			}
			return nil
		},
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) Lookup1KV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg,
		func(is IPStore) error {
			for i := 0; i < num1KElements; i++ {
				err := is.AddNetwork(ib.v6Networks[i%num1KElements])
				if err != nil {
					return err
				}
			}
			return nil
		},
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) AddRemove1KV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v4Networks[i%num1KElements])
			is.RemoveNetwork(ib.v4Networks[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) AddRemove1KV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v6Networks[i%num1KElements])
			is.RemoveNetwork(ib.v6Networks[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) RemoveNonExist1KV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveNetwork(ib.v4Networks[i%num1KElements])
			return nil
		})
}

func (ib ipStoreBench) RemoveNonExist1KV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveNetwork(ib.v6Networks[i%num1KElements])
			return nil
		})
}
