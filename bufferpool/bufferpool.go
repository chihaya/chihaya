// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package bufferpool implements a limited-size pool of reusable,
// equally-sized buffers.
package bufferpool

import (
	"bytes"
	"errors"
)

// BufferPool allows one to easily reuse a limited-sized pool of reusable,
// equally sized buffers.
type BufferPool struct {
	bufSize int
	pool    chan *bytes.Buffer
}

// New returns a newly allocated BufferPool with the given pool size
// and buffer size.
func New(size int, bufSize int) *BufferPool {
	return &BufferPool{
		bufSize,
		make(chan *bytes.Buffer, size),
	}
}

// Take is used to obtain a new zeroed buffer. This may or may not have been
// recycled from the pool depending on factors such as pool being empty.
func (pool *BufferPool) Take() (buf *bytes.Buffer) {
	select {
	case buf = <-pool.pool:
		buf.Reset()
	default:
		internalBuf := make([]byte, 0, pool.bufSize)
		buf = bytes.NewBuffer(internalBuf)
	}
	return
}

// Give is used to attempt to add a buffer to the pool. This may or may not
// be added to the pool depending on factors such as the pool being full.
func (pool *BufferPool) Give(buf *bytes.Buffer) error {
	if buf.Len() != pool.bufSize {
		return errors.New("Gave an incorrectly sized buffer to the pool.")
	}

	select {
	case pool.pool <- buf:
		// Everything went smoothly!
	default:
		return errors.New("Gave a buffer to a full pool.")
	}
	return nil
}
