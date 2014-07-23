// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package stats

import (
	"encoding/json"
	"runtime"
)

// BasicMemStats includes a few of the fields from runtime.MemStats suitable for
// general logging.
type BasicMemStats struct {
	// General statistics.
	Alloc      uint64 // bytes allocated and still in use
	TotalAlloc uint64 // bytes allocated (even if freed)
	Sys        uint64 // bytes obtained from system (sum of XxxSys below)
	Lookups    uint64 // number of pointer lookups
	Mallocs    uint64 // number of mallocs
	Frees      uint64 // number of frees

	// Main allocation heap statistics.
	HeapAlloc    uint64 // bytes allocated and still in use
	HeapSys      uint64 // bytes obtained from system
	HeapIdle     uint64 // bytes in idle spans
	HeapInuse    uint64 // bytes in non-idle span
	HeapReleased uint64 // bytes released to the OS
	HeapObjects  uint64 // total number of allocated objects

	// Garbage collector statistics.
	PauseTotalNs uint64
}

type MemStatsWrapper struct {
	basic   *BasicMemStats
	full    *runtime.MemStats
	verbose bool
}

func NewMemStatsWrapper(verbose bool) *MemStatsWrapper {
	stats := &MemStatsWrapper{
		verbose: verbose,
		full:    &runtime.MemStats{},
	}
	if !verbose {
		stats.basic = &BasicMemStats{}
	}
	return stats
}

func (s *MemStatsWrapper) MarshalJSON() ([]byte, error) {
	if s.verbose {
		return json.Marshal(s.full)
	} else {
		return json.Marshal(s.basic)
	}
}

func (s *MemStatsWrapper) Update() {
	runtime.ReadMemStats(s.full)

	if !s.verbose {
		// Gross, but any decent editor can generate this in a couple commands.
		s.basic.Alloc = s.full.Alloc
		s.basic.TotalAlloc = s.full.TotalAlloc
		s.basic.Sys = s.full.Sys
		s.basic.Lookups = s.full.Lookups
		s.basic.Mallocs = s.full.Mallocs
		s.basic.Frees = s.full.Frees
		s.basic.HeapAlloc = s.full.HeapAlloc
		s.basic.HeapSys = s.full.HeapSys
		s.basic.HeapIdle = s.full.HeapIdle
		s.basic.HeapInuse = s.full.HeapInuse
		s.basic.HeapReleased = s.full.HeapReleased
		s.basic.HeapObjects = s.full.HeapObjects
		s.basic.PauseTotalNs = s.full.PauseTotalNs
	}
}
