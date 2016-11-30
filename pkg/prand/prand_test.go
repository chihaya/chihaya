package prand

import (
	"math/rand"
	"sync/atomic"
	"testing"
)

func BenchmarkContainer_GetReturn(b *testing.B) {
	c := New(1024)
	a := uint64(0)

	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		i := int(atomic.AddUint64(&a, 1))
		var r *rand.Rand

		for p.Next() {
			r = c.Get(i)
			c.Return(i)
		}

		_ = r
	})
}
