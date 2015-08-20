// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package stats

import "runtime"

// BasicMemStats includes a few of the fields from runtime.MemStats suitable for
// general logging.
type BasicMemStats struct {
	// General statistics.
	Alloc      uint64 // bytes allocated and still in use
	TotalAlloc uint64 // bytes allocated (even if freed)
	Sys        uint64 // bytes obtained from system (sum of XxxSys in runtime)
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
	PauseTotalNs  uint64
	LatestPauseNs uint64
}

type memStatsPlaceholder interface{}

// MemStatsWrapper wraps runtime.MemStats with an optionally less verbose JSON
// representation. The JSON field names correspond exactly to the runtime field
// names to avoid reimplementing the entire struct.
type MemStatsWrapper struct {
	memStatsPlaceholder `json:"Memory"`

	basic *BasicMemStats
	cache *runtime.MemStats
}

func NewMemStatsWrapper(verbose bool) *MemStatsWrapper {
	stats := &MemStatsWrapper{cache: &runtime.MemStats{}}

	if verbose {
		stats.memStatsPlaceholder = stats.cache
	} else {
		stats.basic = &BasicMemStats{}
		stats.memStatsPlaceholder = stats.basic
	}
	return stats
}

// Update fetches the current memstats from runtime and resets the cache.
func (s *MemStatsWrapper) Update() {
	runtime.ReadMemStats(s.cache)

	if s.basic != nil {
		// Gross, but any decent editor can generate this in a couple commands.
		s.basic.Alloc = s.cache.Alloc
		s.basic.TotalAlloc = s.cache.TotalAlloc
		s.basic.Sys = s.cache.Sys
		s.basic.Lookups = s.cache.Lookups
		s.basic.Mallocs = s.cache.Mallocs
		s.basic.Frees = s.cache.Frees
		s.basic.HeapAlloc = s.cache.HeapAlloc
		s.basic.HeapSys = s.cache.HeapSys
		s.basic.HeapIdle = s.cache.HeapIdle
		s.basic.HeapInuse = s.cache.HeapInuse
		s.basic.HeapReleased = s.cache.HeapReleased
		s.basic.HeapObjects = s.cache.HeapObjects
		s.basic.PauseTotalNs = s.cache.PauseTotalNs
		s.basic.LatestPauseNs = s.cache.PauseNs[(s.cache.NumGC+255)%256]
	}
}
