// Package xorshift implements the XORShift PRNG.
package xorshift

import "sync"

// XORShift describes the functionality of an XORShift PRNG.
type XORShift interface {
	Next() uint64
}

// XORShift128Plus holds the state of an XORShift128Plus PRNG.
type XORShift128Plus struct {
	state [2]uint64
}

// Next generates a pseudorandom number and advances the state of s.
func (s *XORShift128Plus) Next() uint64 {
	s1 := s.state[0]
	s0 := s.state[1]
	s1Tmp := s1 // need this for result computation
	s.state[0] = s0
	s1 ^= (s1 << 23)                              // a
	s.state[1] = s1 ^ s0 ^ (s1 >> 18) ^ (s0 >> 5) // b, c
	return s0 + s1Tmp
}

// NewXORShift128Plus creates a new XORShift PRNG.
func NewXORShift128Plus(s0, s1 uint64) *XORShift128Plus {
	return &XORShift128Plus{
		state: [2]uint64{s0, s1},
	}
}

// LockedXORShift128Plus is a thread-safe XORShift128Plus.
type LockedXORShift128Plus struct {
	sync.Mutex
	state [2]uint64
}

// NewLockedXORShift128Plus creates a new LockedXORShift128Plus.
func NewLockedXORShift128Plus(s0, s1 uint64) *LockedXORShift128Plus {
	return &LockedXORShift128Plus{
		state: [2]uint64{s0, s1},
	}
}

// Next generates a pseudorandom number and advances the state of s.
func (s *LockedXORShift128Plus) Next() uint64 {
	s.Lock()
	s1 := s.state[0]
	s0 := s.state[1]
	s1Tmp := s1 // need this for result computation
	s.state[0] = s0
	s1 ^= (s1 << 23)                              // a
	s.state[1] = s1 ^ s0 ^ (s1 >> 18) ^ (s0 >> 5) // b, c
	s.Unlock()
	return s0 + s1Tmp
}

// Intn generates an int k that satisfies k >= 0 && k < n.
// n must be > 0.
func Intn(s XORShift, n int) int {
	if n <= 0 {
		panic("invalid n <= 0")
	}
	v := int(s.Next())
	if v < 0 {
		v = -v
	}
	return v % n
}
