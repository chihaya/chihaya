// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"fmt"
	"math/rand"
	"net"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/pkg/random"
)

const num1KElements = 1000

// StringStoreBenchmarker is a collection of benchmarks for StringStore drivers.
//
// All benchmarks can be run in parallel. Parallelism can be controlled using
// the -cpu flag.
//
// Every run of every benchmark expects a new, clean storage. Every benchmark
// should be called with a DriverConfig that ensures this and, if necessary,
// clean up after calling the benchmark. Clean-up time is not considered in the
// benchmark result.
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

	driver         StringStoreDriver
	goroutineShift int
}

func generateLongStrings() (a [num1KElements]string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	strings := make(map[string]struct{})
	for len(strings) < num1KElements {
		strings[random.AlphaNumericString(r, 1000)] = struct{}{}
	}

	i := 0
	for s := range strings {
		a[i] = s
		i++
	}

	return
}

func generateShortStrings() (a [num1KElements]string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	strings := make(map[string]struct{})
	for len(strings) < num1KElements {
		strings[random.AlphaNumericString(r, 10)] = struct{}{}
	}

	i := 0
	for s := range strings {
		a[i] = s
		i++
	}

	return
}

// PrepareStringStoreBenchmarker prepares a reusable suite for StringStore driver
// benchmarks.
func PrepareStringStoreBenchmarker(driver StringStoreDriver) StringStoreBenchmarker {
	return &stringStoreBench{
		sShort:         generateShortStrings(),
		sLong:          generateLongStrings(),
		driver:         driver,
		goroutineShift: num1KElements / runtime.NumCPU(),
	}
}

type stringStoreSetupFunc func(StringStore) error

func stringStoreSetupNOP(StringStore) error { return nil }

func (sb *stringStoreBench) setupStrings(ss StringStore) error {
	for i := 0; i < num1KElements; i++ {
		err := ss.PutString(sb.sShort[i])
		if err != nil {
			return err
		}

		err = ss.PutString(sb.sLong[i])
		if err != nil {
			return err
		}
	}
	return nil
}

type stringStoreBenchFunc func(StringStore, int) error

func (sb *stringStoreBench) runBenchmark(b *testing.B, cfg *DriverConfig, setup stringStoreSetupFunc, execute stringStoreBenchFunc) {
	ss, err := sb.driver.New(cfg)
	require.Nil(b, err, "Constructor error must be nil")
	require.NotNil(b, ss, "StringStore must not be nil")

	err = setup(ss)
	require.Nil(b, err, "Benchmark setup must not fail")

	numGoroutine := int32(0)

	b.ResetTimer()
	b.RunParallel(func(ppb *testing.PB) {
		g := int(atomic.AddInt32(&numGoroutine, 1)) * sb.goroutineShift
		for i := g; ppb.Next(); i++ {
			execute(ss, i)
		}
	})
	b.StopTimer()

	errChan := ss.Stop()
	err = <-errChan
	require.Nil(b, err, "StringStore shutdown must not fail")
}

func (sb *stringStoreBench) AddShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sShort[0])
			return nil
		})
}

func (sb *stringStoreBench) AddLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sLong[0])
			return nil
		})
}

func (sb *stringStoreBench) Add1KShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sShort[i%num1KElements])
			return nil
		})
}

func (sb *stringStoreBench) Add1KLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sLong[i%num1KElements])
			return nil
		})
}

func (sb *stringStoreBench) LookupShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, sb.setupStrings,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sShort[0])
			return nil
		})
}

func (sb *stringStoreBench) LookupLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, sb.setupStrings,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sLong[0])
			return nil
		})
}

func (sb *stringStoreBench) Lookup1KShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, sb.setupStrings,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sShort[i%num1KElements])
			return nil
		})
}

func (sb *stringStoreBench) Lookup1KLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, sb.setupStrings,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sLong[i%num1KElements])
			return nil
		})
}

func (sb *stringStoreBench) AddRemoveShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sShort[0])
			ss.RemoveString(sb.sShort[0])
			return nil
		})
}

func (sb *stringStoreBench) AddRemoveLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sLong[0])
			ss.RemoveString(sb.sLong[0])
			return nil
		})
}

func (sb *stringStoreBench) AddRemove1KShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sShort[i%num1KElements])
			ss.RemoveString(sb.sShort[i%num1KElements])
			return nil
		})
}

func (sb *stringStoreBench) AddRemove1KLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.PutString(sb.sLong[i%num1KElements])
			ss.RemoveString(sb.sLong[i%num1KElements])
			return nil
		})
}

func (sb *stringStoreBench) LookupNonExistShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sShort[0])
			return nil
		})
}

func (sb *stringStoreBench) LookupNonExistLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sLong[0])
			return nil
		})
}

func (sb *stringStoreBench) LookupNonExist1KShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sShort[i%num1KElements])
			return nil
		})
}

func (sb *stringStoreBench) LookupNonExist1KLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.HasString(sb.sLong[i%num1KElements])
			return nil
		})
}

func (sb *stringStoreBench) RemoveNonExistShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.RemoveString(sb.sShort[0])
			return nil
		})
}

func (sb *stringStoreBench) RemoveNonExistLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.RemoveString(sb.sLong[0])
			return nil
		})
}

func (sb *stringStoreBench) RemoveNonExist1KShort(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.RemoveString(sb.sShort[i%num1KElements])
			return nil
		})
}

func (sb *stringStoreBench) RemoveNonExist1KLong(b *testing.B, cfg *DriverConfig) {
	sb.runBenchmark(b, cfg, stringStoreSetupNOP,
		func(ss StringStore, i int) error {
			ss.RemoveString(sb.sLong[i%num1KElements])
			return nil
		})
}

// IPStoreBenchmarker is a collection of benchmarks for IPStore drivers.
//
// All benchmarks can be run in parallel. Parallelism can be controlled using
// the -cpu flag.
//
// Every run of every benchmark expects a new, clean storage. Every benchmark
// should be called with a DriverConfig that ensures this and, if necessary,
// clean up after calling the benchmark. Clean-up time is not considered in the
// benchmark result.
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

	driver         IPStoreDriver
	goroutineShift int
}

// PrepareIPStoreBenchmarker prepares a reusable suite for StringStore driver
// benchmarks.
func PrepareIPStoreBenchmarker(driver IPStoreDriver) IPStoreBenchmarker {
	return &ipStoreBench{
		v4IPs:          generateV4IPs(),
		v6IPs:          generateV6IPs(),
		v4Networks:     generateV4Networks(),
		v6Networks:     generateV6Networks(),
		driver:         driver,
		goroutineShift: num1KElements / runtime.NumCPU(),
	}
}

type ipStoreSetupFunc func(IPStore) error

func ipStoreSetupNOP(IPStore) error { return nil }

func (ib *ipStoreBench) setupIPs(is IPStore) error {
	for i := 0; i < num1KElements; i++ {
		err := is.AddIP(ib.v4IPs[i])
		if err != nil {
			return err
		}

		err = is.AddIP(ib.v6IPs[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (ib *ipStoreBench) setupNetworks(is IPStore) error {
	for i := 0; i < num1KElements; i++ {
		err := is.AddNetwork(ib.v4Networks[i])
		if err != nil {
			return err
		}

		err = is.AddNetwork(ib.v6Networks[i])
		if err != nil {
			return err
		}
	}
	return nil
}

type ipStoreBenchFunc func(IPStore, int) error

func (ib *ipStoreBench) runBenchmark(b *testing.B, cfg *DriverConfig, setup ipStoreSetupFunc, execute ipStoreBenchFunc) {
	is, err := ib.driver.New(cfg)
	require.Nil(b, err, "Constructor error must be nil")
	require.NotNil(b, is, "IPStore must not be nil")

	err = setup(is)
	require.Nil(b, err, "Benchmark setup must not fail")

	numGoroutine := int32(0)

	b.ResetTimer()
	b.RunParallel(func(ppb *testing.PB) {
		g := int(atomic.AddInt32(&numGoroutine, 1)) * ib.goroutineShift
		for i := g; ppb.Next(); i++ {
			execute(is, i)
		}
	})
	b.StopTimer()

	errChan := is.Stop()
	err = <-errChan
	require.Nil(b, err, "IPStore shutdown must not fail")
}

func (ib *ipStoreBench) AddV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v4IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) AddV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v6IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) LookupV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ib.setupIPs,
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) LookupV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ib.setupIPs,
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) AddRemoveV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v4IPs[0])
			is.RemoveIP(ib.v4IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) AddRemoveV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v6IPs[0])
			is.RemoveIP(ib.v6IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) LookupNonExistV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) LookupNonExistV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) RemoveNonExistV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveIP(ib.v4IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) RemoveNonExistV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveIP(ib.v6IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) AddV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v4Networks[0])
			return nil
		})
}

func (ib *ipStoreBench) AddV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v6Networks[0])
			return nil
		})
}

func (ib *ipStoreBench) LookupV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ib.setupNetworks,
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) LookupV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ib.setupNetworks,
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[0])
			return nil
		})
}

func (ib *ipStoreBench) AddRemoveV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v4Networks[0])
			is.RemoveNetwork(ib.v4Networks[0])
			return nil
		})
}

func (ib *ipStoreBench) AddRemoveV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v6Networks[0])
			is.RemoveNetwork(ib.v6Networks[0])
			return nil
		})
}

func (ib *ipStoreBench) RemoveNonExistV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveNetwork(ib.v4Networks[0])
			return nil
		})
}

func (ib *ipStoreBench) RemoveNonExistV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveNetwork(ib.v6Networks[0])
			return nil
		})
}

func (ib *ipStoreBench) Add1KV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) Add1KV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) Lookup1KV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ib.setupIPs,
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) Lookup1KV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ib.setupIPs,
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) AddRemove1KV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v4IPs[i%num1KElements])
			is.RemoveIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) AddRemove1KV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddIP(ib.v6IPs[i%num1KElements])
			is.RemoveIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) LookupNonExist1KV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) LookupNonExist1KV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) RemoveNonExist1KV4(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) RemoveNonExist1KV6(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) Add1KV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v4Networks[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) Add1KV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v6Networks[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) Lookup1KV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ib.setupNetworks,
		func(is IPStore, i int) error {
			is.HasIP(ib.v4IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) Lookup1KV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ib.setupNetworks,
		func(is IPStore, i int) error {
			is.HasIP(ib.v6IPs[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) AddRemove1KV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v4Networks[i%num1KElements])
			is.RemoveNetwork(ib.v4Networks[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) AddRemove1KV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.AddNetwork(ib.v6Networks[i%num1KElements])
			is.RemoveNetwork(ib.v6Networks[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) RemoveNonExist1KV4Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveNetwork(ib.v4Networks[i%num1KElements])
			return nil
		})
}

func (ib *ipStoreBench) RemoveNonExist1KV6Network(b *testing.B, cfg *DriverConfig) {
	ib.runBenchmark(b, cfg, ipStoreSetupNOP,
		func(is IPStore, i int) error {
			is.RemoveNetwork(ib.v6Networks[i%num1KElements])
			return nil
		})
}

// PeerStoreBenchmarker is a collection of benchmarks for PeerStore drivers.
//
// All benchmarks can be run in parallel. Parallelism can be controlled using
// the -cpu flag.
//
// Every run of every benchmark expects a new, clean storage. Every benchmark
// should be called with a DriverConfig that ensures this and, if necessary,
// clean up after calling the benchmark. Clean-up time is not considered in the
// benchmark result.
type PeerStoreBenchmarker interface {
	PutSeeder(*testing.B, *DriverConfig)
	PutSeeder1KInfohash(*testing.B, *DriverConfig)
	PutSeeder1KSeeders(*testing.B, *DriverConfig)
	PutSeeder1KInfohash1KSeeders(*testing.B, *DriverConfig)

	PutDeleteSeeder(*testing.B, *DriverConfig)
	PutDeleteSeeder1KInfohash(*testing.B, *DriverConfig)
	PutDeleteSeeder1KSeeders(*testing.B, *DriverConfig)
	PutDeleteSeeder1KInfohash1KSeeders(*testing.B, *DriverConfig)

	DeleteSeederNonExist(*testing.B, *DriverConfig)
	DeleteSeederNonExist1KInfohash(*testing.B, *DriverConfig)
	DeleteSeederNonExist1KSeeders(*testing.B, *DriverConfig)
	DeleteSeederNonExist1KInfohash1KSeeders(*testing.B, *DriverConfig)

	PutGraduateDeleteLeecher(*testing.B, *DriverConfig)
	PutGraduateDeleteLeecher1KInfohash(*testing.B, *DriverConfig)
	PutGraduateDeleteLeecher1KLeechers(*testing.B, *DriverConfig)
	PutGraduateDeleteLeecher1KInfohash1KLeechers(*testing.B, *DriverConfig)

	GraduateLeecherNonExist(*testing.B, *DriverConfig)
	GraduateLeecherNonExist1KInfohash(*testing.B, *DriverConfig)
	GraduateLeecherNonExist1KLeechers(*testing.B, *DriverConfig)
	GraduateLeecherNonExist1KInfohash1KLeechers(*testing.B, *DriverConfig)

	AnnouncePeers(*testing.B, *DriverConfig)
	AnnouncePeers1KInfohash(*testing.B, *DriverConfig)
	AnnouncePeersSeeder(*testing.B, *DriverConfig)
	AnnouncePeersSeeder1KInfohash(*testing.B, *DriverConfig)

	GetSeeders(*testing.B, *DriverConfig)
	GetSeeders1KInfohash(*testing.B, *DriverConfig)

	NumSeeders(*testing.B, *DriverConfig)
	NumSeeders1KInfohash(*testing.B, *DriverConfig)
}

type peerStoreBench struct {
	infohashes     [num1KElements]chihaya.InfoHash
	peers          [num1KElements]chihaya.Peer
	driver         PeerStoreDriver
	goroutineShift int
}

func generateInfohashes() (a [num1KElements]chihaya.InfoHash) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	strings := make(map[string]struct{})

	for len(strings) < num1KElements {
		strings[random.AlphaNumericString(r, 20)] = struct{}{}
	}

	i := 0
	for s := range strings {
		a[i] = chihaya.InfoHashFromString(s)
		i++
	}

	return
}

func generatePeers() (a [num1KElements]chihaya.Peer) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	minPort := 20000
	maxPort := 60000

	strings := make(map[string]struct{})
	ips := make(map[string]struct{})
	ports := make(map[uint16]struct{})

	for len(strings) < num1KElements {
		strings[random.AlphaNumericString(r, 20)] = struct{}{}
	}

	for len(ips) < num1KElements {
		b := make([]byte, 4)
		ip := net.IP(b)

		for i := range ip {
			b := r.Intn(254) + 1
			ip[i] = byte(b)
		}

		ips[ip.String()] = struct{}{}
	}

	for len(ports) < num1KElements {
		ports[uint16(r.Int63()%int64(maxPort-minPort))+uint16(minPort)] = struct{}{}
	}

	i := 0
	for s := range strings {
		a[i] = chihaya.Peer{
			ID: chihaya.PeerIDFromString(s),
		}
		i++
	}

	i = 0
	for ip := range ips {
		a[i].IP = net.ParseIP(ip)
		i++
	}

	i = 0
	for p := range ports {
		a[i].Port = p
		i++
	}

	return
}

// PreparePeerStoreBenchmarker prepares a reusable suite for PeerStore driver
// benchmarks.
func PreparePeerStoreBenchmarker(driver PeerStoreDriver) PeerStoreBenchmarker {
	return &peerStoreBench{
		driver:         driver,
		infohashes:     generateInfohashes(),
		peers:          generatePeers(),
		goroutineShift: num1KElements / runtime.NumCPU(),
	}
}

type peerStoreSetupFunc func(PeerStore) error

func peerStoreSetupNOP(PeerStore) error { return nil }

func (pb *peerStoreBench) setupPeers(ps PeerStore) error {
	for i := 0; i < num1KElements; i++ {
		for j := 0; j < num1KElements; j++ {
			var err error
			if j < num1KElements/2 {
				err = ps.PutLeecher(pb.infohashes[i], pb.peers[j])
			} else {
				err = ps.PutSeeder(pb.infohashes[i], pb.peers[j])
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type peerStoreBenchFunc func(PeerStore, int) error

func (pb *peerStoreBench) runBenchmark(b *testing.B, cfg *DriverConfig, setup peerStoreSetupFunc, execute peerStoreBenchFunc) {
	ps, err := pb.driver.New(cfg)
	require.Nil(b, err, "Constructor error must be nil")
	require.NotNil(b, ps, "PeerStore must not be nil")

	err = setup(ps)
	require.Nil(b, err, "Benchmark setup must not fail")

	numGoroutine := int32(0)

	b.ResetTimer()
	b.RunParallel(func(ppb *testing.PB) {
		g := int(atomic.AddInt32(&numGoroutine, 1)) * pb.goroutineShift
		for i := g; ppb.Next(); i++ {
			execute(ps, i)
		}
	})
	b.StopTimer()

	errChan := ps.Stop()
	err = <-errChan
	require.Nil(b, err, "PeerStore shutdown must not fail")
}

func (pb *peerStoreBench) PutSeeder(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutSeeder(pb.infohashes[0], pb.peers[0])
			return nil
		})
}

func (pb *peerStoreBench) PutSeeder1KInfohash(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutSeeder(pb.infohashes[i%num1KElements], pb.peers[0])
			return nil
		})
}

func (pb *peerStoreBench) PutSeeder1KSeeders(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutSeeder(pb.infohashes[0], pb.peers[i%num1KElements])
			return nil
		})
}

func (pb *peerStoreBench) PutSeeder1KInfohash1KSeeders(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutSeeder(pb.infohashes[i%num1KElements], pb.peers[(i*3)%num1KElements])
			return nil
		})
}

func (pb *peerStoreBench) PutDeleteSeeder(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutSeeder(pb.infohashes[0], pb.peers[0])
			ps.DeleteSeeder(pb.infohashes[0], pb.peers[0])
			return nil
		})
}

func (pb *peerStoreBench) PutDeleteSeeder1KInfohash(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutSeeder(pb.infohashes[i%num1KElements], pb.peers[0])
			ps.DeleteSeeder(pb.infohashes[i%num1KElements], pb.peers[0])
			return nil
		})
}

func (pb *peerStoreBench) PutDeleteSeeder1KSeeders(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutSeeder(pb.infohashes[0], pb.peers[i%num1KElements])
			ps.DeleteSeeder(pb.infohashes[0], pb.peers[i%num1KElements])
			return nil
		})
}

func (pb *peerStoreBench) PutDeleteSeeder1KInfohash1KSeeders(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutSeeder(pb.infohashes[i%num1KElements], pb.peers[(i*3)%num1KElements])
			ps.DeleteSeeder(pb.infohashes[i%num1KElements], pb.peers[(i*3)%num1KElements])
			return nil
		})
}

func (pb *peerStoreBench) DeleteSeederNonExist(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.DeleteSeeder(pb.infohashes[0], pb.peers[0])
			return nil
		})
}

func (pb *peerStoreBench) DeleteSeederNonExist1KInfohash(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.DeleteSeeder(pb.infohashes[i%num1KElements], pb.peers[0])
			return nil
		})
}

func (pb *peerStoreBench) DeleteSeederNonExist1KSeeders(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.DeleteSeeder(pb.infohashes[0], pb.peers[i%num1KElements])
			return nil
		})
}

func (pb *peerStoreBench) DeleteSeederNonExist1KInfohash1KSeeders(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.DeleteSeeder(pb.infohashes[i%num1KElements], pb.peers[(i*3)%num1KElements])
			return nil
		})
}

func (pb *peerStoreBench) GraduateLeecherNonExist(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.GraduateLeecher(pb.infohashes[0], pb.peers[0])
			return nil
		})
}

func (pb *peerStoreBench) GraduateLeecherNonExist1KInfohash(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.GraduateLeecher(pb.infohashes[i%num1KElements], pb.peers[0])
			return nil
		})
}

func (pb *peerStoreBench) GraduateLeecherNonExist1KLeechers(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.GraduateLeecher(pb.infohashes[0], pb.peers[i%num1KElements])
			return nil
		})
}

func (pb *peerStoreBench) GraduateLeecherNonExist1KInfohash1KLeechers(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.GraduateLeecher(pb.infohashes[i%num1KElements], pb.peers[(i*3)%num1KElements])
			return nil
		})
}

func (pb *peerStoreBench) PutGraduateDeleteLeecher(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutLeecher(pb.infohashes[0], pb.peers[0])
			ps.GraduateLeecher(pb.infohashes[0], pb.peers[0])
			ps.DeleteSeeder(pb.infohashes[0], pb.peers[0])
			return nil
		})
}

func (pb *peerStoreBench) PutGraduateDeleteLeecher1KInfohash(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutLeecher(pb.infohashes[i%num1KElements], pb.peers[0])
			ps.GraduateLeecher(pb.infohashes[i%num1KElements], pb.peers[0])
			ps.DeleteSeeder(pb.infohashes[i%num1KElements], pb.peers[0])
			return nil
		})
}

func (pb *peerStoreBench) PutGraduateDeleteLeecher1KLeechers(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutLeecher(pb.infohashes[0], pb.peers[i%num1KElements])
			ps.GraduateLeecher(pb.infohashes[0], pb.peers[i%num1KElements])
			ps.DeleteSeeder(pb.infohashes[0], pb.peers[i%num1KElements])
			return nil
		})
}

func (pb *peerStoreBench) PutGraduateDeleteLeecher1KInfohash1KLeechers(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, peerStoreSetupNOP,
		func(ps PeerStore, i int) error {
			ps.PutLeecher(pb.infohashes[i%num1KElements], pb.peers[(i*3)%num1KElements])
			ps.GraduateLeecher(pb.infohashes[i%num1KElements], pb.peers[(i*3)%num1KElements])
			ps.DeleteSeeder(pb.infohashes[i%num1KElements], pb.peers[(i*3)%num1KElements])
			return nil
		})
}

func (pb *peerStoreBench) AnnouncePeers(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, pb.setupPeers,
		func(ps PeerStore, i int) error {
			ps.AnnouncePeers(pb.infohashes[0], false, 50, pb.peers[0], chihaya.Peer{})
			return nil
		})
}

func (pb *peerStoreBench) AnnouncePeers1KInfohash(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, pb.setupPeers,
		func(ps PeerStore, i int) error {
			ps.AnnouncePeers(pb.infohashes[i%num1KElements], false, 50, pb.peers[0], chihaya.Peer{})
			return nil
		})
}

func (pb *peerStoreBench) AnnouncePeersSeeder(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, pb.setupPeers,
		func(ps PeerStore, i int) error {
			ps.AnnouncePeers(pb.infohashes[0], true, 50, pb.peers[0], chihaya.Peer{})
			return nil
		})
}

func (pb *peerStoreBench) AnnouncePeersSeeder1KInfohash(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, pb.setupPeers,
		func(ps PeerStore, i int) error {
			ps.AnnouncePeers(pb.infohashes[i%num1KElements], true, 50, pb.peers[0], chihaya.Peer{})
			return nil
		})
}

func (pb *peerStoreBench) GetSeeders(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, pb.setupPeers,
		func(ps PeerStore, i int) error {
			ps.GetSeeders(pb.infohashes[0])
			return nil
		})
}

func (pb *peerStoreBench) GetSeeders1KInfohash(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, pb.setupPeers,
		func(ps PeerStore, i int) error {
			ps.GetSeeders(pb.infohashes[i%num1KElements])
			return nil
		})
}

func (pb *peerStoreBench) NumSeeders(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, pb.setupPeers,
		func(ps PeerStore, i int) error {
			ps.NumSeeders(pb.infohashes[0])
			return nil
		})
}

func (pb *peerStoreBench) NumSeeders1KInfohash(b *testing.B, cfg *DriverConfig) {
	pb.runBenchmark(b, cfg, pb.setupPeers,
		func(ps PeerStore, i int) error {
			ps.NumSeeders(pb.infohashes[i%num1KElements])
			return nil
		})
}
