package xorshift

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIntn(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	s := NewXORShift128Plus(rand.Uint64(), rand.Uint64())
	for i := 0; i < 10000; i++ {
		k := Intn(s, 10)
		require.True(t, k >= 0, "Intn() must be >= 0")
		require.True(t, k < 10, "Intn(k) must be < k")
	}
}

func BenchmarkXORShift128Plus_Next(b *testing.B) {
	s := NewXORShift128Plus(rand.Uint64(), rand.Uint64())
	var k uint64
	for i := 0; i < b.N; i++ {
		k = s.Next()
	}
	_ = k
}

func BenchmarkIntnXORShift128Plus(b *testing.B) {
	s := NewXORShift128Plus(rand.Uint64(), rand.Uint64())
	var k int
	for i := 0; i < b.N; i++ {
		k = Intn(s, 1000)
	}
	_ = k
}

func BenchmarkLockedXORShift128Plus_Next(b *testing.B) {
	s := NewLockedXORShift128Plus(rand.Uint64(), rand.Uint64())
	var k uint64
	for i := 0; i < b.N; i++ {
		k = s.Next()
	}
	_ = k
}
