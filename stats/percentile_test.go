package stats

import (
	"math/rand"
	"testing"
	"time"
)

func TestPercentiles(t *testing.T) {
	rand.Seed(time.Now().Unix())

	testUniformRandom(t, 1, 0.5)
	testUniformRandom(t, 1, 0.9)
	testUniformRandom(t, 1, 0.95)
	testUniformRandom(t, 10000, 0.5)
	testUniformRandom(t, 10000, 0.9)
	testUniformRandom(t, 10000, 0.95)
}

func testUniformRandom(t *testing.T, max, percentile float64) {
	p := NewPercentile(percentile, 256)

	for i := 0; i < 100000; i++ {
		p.AddSample(rand.Float64() * max)
	}

	got := p.Value()
	expected := percentile * max
	maxError := 0.01

	if got < expected*(1-maxError) || got > expected*(1+maxError) {
		t.Errorf("Percentile out of range\n  actual: %f\nexpected: %f\n   error: %f%%\n", got, expected, (got-expected)/expected*100)
	}
}

func BenchmarkPercentiles64(b *testing.B) {
	benchmarkUniformRandom(b, 64, 0.5)
}

func BenchmarkPercentiles128(b *testing.B) {
	benchmarkUniformRandom(b, 128, 0.5)
}

func BenchmarkPercentiles256(b *testing.B) {
	benchmarkUniformRandom(b, 256, 0.5)
}

func BenchmarkPercentiles512(b *testing.B) {
	benchmarkUniformRandom(b, 512, 0.5)
}

func benchmarkUniformRandom(b *testing.B, window int, percentile float64) {
	p := NewPercentile(percentile, window)

	numbers := make([]float64, b.N)

	for i := 0; i < b.N; i++ {
		numbers[i] = rand.Float64()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.AddSample(numbers[i])
	}
}
