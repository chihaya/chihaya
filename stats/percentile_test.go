package stats

import (
	"testing"
	"math/rand"
)

func TestPercentiles(t *testing.T) {
	testInRange(t, 1, 0.5)
	testInRange(t, 1, 0.9)
	testInRange(t, 1, 0.95)
	testInRange(t, 10000, 0.5)
	testInRange(t, 10000, 0.9)
	testInRange(t, 10000, 0.95)
}

func testInRange(t *testing.T, max, percentile float64) {
	p := NewPercentile(percentile, 10)

	for i := 0; i < 1000; i++ {
		p.AddSample(rand.Float64() * max)
	}

	got := p.Value()
	expected := percentile * max

	if got < expected * (1 - 0.02) || got > expected * (1 + 0.02) {
		t.Errorf("Percentile out of range\n  actual: %f\nexpected: %f", got, expected)
	}
}
