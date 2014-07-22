package stats

import (
	"sort"
)

type Percentile struct {
	percentile float64
	values sort.Float64Slice
	offset int
}

func NewPercentile(percentile float64, sampleWindow int) *Percentile {
	return &Percentile{
		percentile: percentile,
		values: make([]float64, 0, sampleWindow),
	}
}

func (p *Percentile) AddSample(sample float64) {
	p.values = append(p.values, sample)
	sort.Sort(p.values)
}

func (p *Percentile) Value() float64 {
	if len(p.values) == 0 {
		return 0
	}

	return p.values[round(p.index())]
}

func (p *Percentile) index() float64 {
	return float64(len(p.values)) * p.percentile - float64(p.offset)
}

func round(value float64) int64 {
	if value < 0.0 {
		value -= 0.5
	} else {
		value += 0.5
	}

	return int64(value)
}
