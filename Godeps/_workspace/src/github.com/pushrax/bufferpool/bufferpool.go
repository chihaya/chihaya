// Copyright 2013 The Bufferpool Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package bufferpool implements a capacity-limited pool of reusable,
// equally-sized buffers.
package bufferpool

import (
	"bytes"
	"errors"
)

// A BufferPool is a capacity-limited pool of equally sized buffers.
type BufferPool struct {
	bufferSize int
	pool       chan []byte
}

// New returns a newly allocated BufferPool with the given maximum pool size
// and buffer size.
func New(poolSize, bufferSize int) *BufferPool {
	return &BufferPool{
		bufferSize,
		make(chan []byte, poolSize),
	}
}

// Take is used to obtain a new zeroed buffer. This will allocate a new buffer
// if the pool was empty.
func (pool *BufferPool) Take() *bytes.Buffer {
	return bytes.NewBuffer(pool.TakeSlice())
}

// TakeSlice is used to obtain a new slice. This will allocate a new slice
// if the pool was empty.
func (pool *BufferPool) TakeSlice() (slice []byte) {
	select {
	case slice = <-pool.pool:
	default:
		slice = make([]byte, 0, pool.bufferSize)
	}
	return
}

// Give is used to attempt to return a buffer to the pool. It may not
// be added to the pool if it was already full.
func (pool *BufferPool) Give(buf *bytes.Buffer) error {
	if buf.Len() != pool.bufferSize {
		return errors.New("Gave an incorrectly sized buffer to the pool.")
	}

	buf.Reset()
	slice := buf.Bytes()
	return pool.GiveSlice(slice[:buf.Len()])
}

// GiveSlice is used to attempt to return a slice to the pool. It may not
// be added to the pool if it was already full.
func (pool *BufferPool) GiveSlice(slice []byte) error {
	select {
	case pool.pool <- slice:
		// Everything went smoothly!
	default:
		return errors.New("Gave a buffer to a full pool.")
	}
	return nil
}
