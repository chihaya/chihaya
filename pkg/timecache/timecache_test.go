package timecache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	c := New()
	require.NotNil(t, c)

	now := c.Now()
	require.False(t, now.IsZero())

	nsec := c.NowUnixNano()
	require.NotEqual(t, 0, nsec)

	sec := c.NowUnix()
	require.NotEqual(t, 0, sec)
}

func TestRunStop(t *testing.T) {
	c := New()
	require.NotNil(t, c)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Run(1 * time.Second)
	}()

	c.Stop()

	wg.Wait()
}

func TestMultipleStop(t *testing.T) {
	c := New()
	require.NotNil(t, c)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Run(1 * time.Second)
	}()

	c.Stop()
	c.Stop()

	wg.Wait()
}

func doBenchmark(b *testing.B, f func(tc *TimeCache) func(*testing.PB)) {
	tc := New()
	require.NotNil(b, tc)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		tc.Run(1 * time.Second)
	}()

	b.RunParallel(f(tc))

	tc.Stop()
	wg.Wait()
}

func BenchmarkNow(b *testing.B) {
	doBenchmark(b, func(tc *TimeCache) func(pb *testing.PB) {
		return func(pb *testing.PB) {
			var now time.Time
			for pb.Next() {
				now = tc.Now()
			}
			_ = now
		}
	})
}

func BenchmarkNowUnix(b *testing.B) {
	doBenchmark(b, func(tc *TimeCache) func(pb *testing.PB) {
		return func(pb *testing.PB) {
			var now int64
			for pb.Next() {
				now = tc.NowUnix()
			}
			_ = now
		}
	})
}

func BenchmarkNowUnixNano(b *testing.B) {
	doBenchmark(b, func(tc *TimeCache) func(pb *testing.PB) {
		return func(pb *testing.PB) {
			var now int64
			for pb.Next() {
				now = tc.NowUnixNano()
			}
			_ = now
		}
	})
}

func BenchmarkNowGlobal(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var now time.Time
		for pb.Next() {
			now = Now()
		}
		_ = now
	})
}

func BenchmarkNowUnixGlobal(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var now int64
		for pb.Next() {
			now = NowUnix()
		}
		_ = now
	})
}

func BenchmarkNowUnixNanoGlobal(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var now int64
		for pb.Next() {
			now = NowUnixNano()
		}
		_ = now
	})
}

func BenchmarkTimeNow(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var now time.Time
		for pb.Next() {
			now = time.Now()
		}
		_ = now
	})
}
