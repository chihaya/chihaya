// Package timecache provides a cache for the system clock, to avoid calls to
// time.Now().
// The time is stored as one int64 which holds the number of nanoseconds since
// the Unix Epoch. The value is accessed using atomic primitives, without
// locking.
// The package runs a global singleton TimeCache that is is updated every
// second.
package timecache

import (
	"sync"
	"sync/atomic"
	"time"
)

// t is the global TimeCache.
var t *TimeCache

func init() {
	t = &TimeCache{
		clock:   time.Now().UnixNano(),
		closed:  make(chan struct{}),
		running: make(chan struct{}),
	}

	go t.Run(1 * time.Second)
}

// A TimeCache is a cache for the current system time.
// The cached time has nanosecond precision.
type TimeCache struct {
	// clock saves the current time's nanoseconds since the Epoch.
	// Must be accessed atomically.
	clock int64

	closed  chan struct{}
	running chan struct{}
	m       sync.Mutex
}

// New returns a new TimeCache instance.
// The TimeCache must be started to update the time.
func New() *TimeCache {
	return &TimeCache{
		clock:   time.Now().UnixNano(),
		closed:  make(chan struct{}),
		running: make(chan struct{}),
	}
}

// Run runs the TimeCache, updating the cached clock value once every interval
// and blocks until Stop is called.
func (t *TimeCache) Run(interval time.Duration) {
	t.m.Lock()
	select {
	case <-t.running:
		panic("Run called multiple times")
	default:
	}
	close(t.running)
	t.m.Unlock()

	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-t.closed:
			tick.Stop()
			return
		case now := <-tick.C:
			atomic.StoreInt64(&t.clock, now.UnixNano())
		}
	}
}

// Stop stops the TimeCache.
// The cached time remains valid but will not be updated anymore.
// A TimeCache can not be restarted. Construct a new one instead.
// Calling Stop again is a no-op.
func (t *TimeCache) Stop() {
	t.m.Lock()
	defer t.m.Unlock()

	select {
	case <-t.closed:
		return
	default:
	}
	close(t.closed)
}

// Now returns the cached time as a time.Time value.
func (t *TimeCache) Now() time.Time {
	return time.Unix(0, atomic.LoadInt64(&t.clock))
}

// NowUnixNano returns the cached time as nanoseconds since the Unix Epoch.
func (t *TimeCache) NowUnixNano() int64 {
	return atomic.LoadInt64(&t.clock)
}

// NowUnix returns the cached time as seconds since the Unix Epoch.
func (t *TimeCache) NowUnix() int64 {
	// Adopted from time.Unix
	nsec := atomic.LoadInt64(&t.clock)
	sec := nsec / 1e9
	nsec -= sec * 1e9
	if nsec < 0 {
		sec--
	}
	return sec
}

// Now calls Now on the global TimeCache instance.
func Now() time.Time {
	return t.Now()
}

// NowUnixNano calls NowUnixNano on the global TimeCache instance.
func NowUnixNano() int64 {
	return t.NowUnixNano()
}

// NowUnix calls NowUnix on the global TimeCache instance.
func NowUnix() int64 {
	return t.NowUnix()
}
