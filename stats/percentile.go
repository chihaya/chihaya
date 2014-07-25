// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package stats

import (
	"encoding/json"
	"math"
	"sort"
	"sync/atomic"
)

// Percentile implements an efficient percentile calculation of
// arbitrary float64 samples.
type Percentile struct {
	percentile float64

	samples int64
	offset  int64

	values []float64
	value  uint64 // These bits are really a float64.
}

// NewPercentile returns a Percentile with a given threshold.
func NewPercentile(percentile float64) *Percentile {
	return &Percentile{
		percentile: percentile,

		// 256 samples is fast, and accurate for most distributions.
		values: make([]float64, 0, 256),
	}
}

// NewPercentileWithWindow returns a Percentile with a given threshold
// and window size (accuracy).
func NewPercentileWithWindow(percentile float64, sampleWindow int) *Percentile {
	return &Percentile{
		percentile: percentile,
		values:     make([]float64, 0, sampleWindow),
	}
}

// Value returns the current value at the stored percentile.
// It is thread-safe, and may be called concurrently with AddSample.
func (p *Percentile) Value() float64 {
	bits := atomic.LoadUint64(&p.value)
	return math.Float64frombits(bits)
}

// AddSample adds a single float64 sample to the data set.
// It is not thread-safe, and not be called concurrently.
func (p *Percentile) AddSample(sample float64) {
	p.samples++

	if len(p.values) == cap(p.values) {
		target := float64(p.samples)*p.percentile - float64(cap(p.values))/2
		offset := round(math.Max(target, 0))

		if sample > p.values[0] {
			if offset > p.offset {
				idx := sort.SearchFloat64s(p.values[1:], sample)
				copy(p.values, p.values[1:idx+1])

				p.values[idx] = sample
				p.offset++
			} else if sample < p.values[len(p.values)-1] {
				idx := sort.SearchFloat64s(p.values, sample)
				copy(p.values[idx+1:], p.values[idx:])

				p.values[idx] = sample
			}
		} else {
			if offset > p.offset {
				p.offset++
			} else {
				copy(p.values[1:], p.values)
				p.values[0] = sample
			}
		}
	} else {
		idx := sort.SearchFloat64s(p.values, sample)
		p.values = p.values[:len(p.values)+1]
		copy(p.values[idx+1:], p.values[idx:])
		p.values[idx] = sample
	}

	bits := math.Float64bits(p.values[p.index()])
	atomic.StoreUint64(&p.value, bits)
}

func (p *Percentile) index() int64 {
	idx := round(float64(p.samples)*p.percentile - float64(p.offset))
	last := int64(len(p.values)) - 1

	if idx > last {
		return last
	}

	return idx
}

func (p *Percentile) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Value())
}

func round(value float64) int64 {
	if value < 0.0 {
		value -= 0.5
	} else {
		value += 0.5
	}

	return int64(value)
}
