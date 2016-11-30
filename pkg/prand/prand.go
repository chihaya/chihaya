// Package prand allows parallel access to randomness based on indices or
// infohashes.
package prand

import (
	"encoding/binary"
	"math/rand"
	"sync"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
)

type lockableRand struct {
	*rand.Rand
	*sync.Mutex
}

// Container is a container for sources of random numbers that can be locked
// individually.
type Container struct {
	rands []lockableRand
}

// NewSeeded returns a new Container with num sources that are seeeded with
// seed.
func NewSeeded(num int, seed int64) *Container {
	toReturn := Container{
		rands: make([]lockableRand, num),
	}

	for i := 0; i < num; i++ {
		toReturn.rands[i].Rand = rand.New(rand.NewSource(seed))
		toReturn.rands[i].Mutex = &sync.Mutex{}
	}

	return &toReturn
}

// New returns a new Container with num sources that are seeded with the current
// time.
func New(num int) *Container {
	return NewSeeded(num, time.Now().UnixNano())
}

// Get locks and returns the nth source.
//
// Get panics if n is not a valid index for this Container.
func (s *Container) Get(n int) *rand.Rand {
	r := s.rands[n]
	r.Lock()
	return r.Rand
}

// GetByInfohash locks and returns a source derived from the infohash.
func (s *Container) GetByInfohash(ih bittorrent.InfoHash) *rand.Rand {
	u := int(binary.BigEndian.Uint32(ih[:4])) % len(s.rands)
	return s.Get(u)
}

// Return returns the nth source to be available again.
//
// Return panics if n is not a valid index for this Container.
// Return also panics if the nth source is unlocked already.
func (s *Container) Return(n int) {
	s.rands[n].Unlock()
}

// ReturnByInfohash returns the source derived from the infohash.
//
// ReturnByInfohash panics if the source is unlocked already.
func (s *Container) ReturnByInfohash(ih bittorrent.InfoHash) {
	u := int(binary.BigEndian.Uint32(ih[:4])) % len(s.rands)
	s.Return(u)
}
