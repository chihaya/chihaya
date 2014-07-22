package stats

import (
	"math"
	"sort"
	"sync/atomic"
	"unsafe"
)

type Percentile struct {
	percentile float64

	samples int64
	offset  int64

	values []float64
	value  *unsafe.Pointer
}

func NewPercentile(percentile float64, sampleWindow int) *Percentile {
	initial := 0
	ptr := unsafe.Pointer(&initial)

	return &Percentile{
		percentile: percentile,

		values: make([]float64, 0, sampleWindow),
		value:  &ptr,
	}
}

func (p *Percentile) AddSample(sample float64) {
	p.samples++

	if p.samples > int64(cap(p.values)) {
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

	value := p.values[p.index()]
	atomic.SwapPointer(p.value, unsafe.Pointer(&value))
}

func (p *Percentile) Value() float64 {
	pointer := atomic.LoadPointer(p.value)
	return *(*float64)(pointer)
}

func (p *Percentile) index() int64 {
	idx := round(float64(p.samples)*p.percentile - float64(p.offset))
	last := int64(len(p.values)) - 1

	if idx > last {
		return last
	}

	return idx
}

func round(value float64) int64 {
	if value < 0.0 {
		value -= 0.5
	} else {
		value += 0.5
	}

	return int64(value)
}
