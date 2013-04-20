// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package util

import (
	"bytes"
)

// BufferPool allows one to easily reuse a limited-sized pool of equally sized
// buffers.
type BufferPool struct {
	bufSize int
	pool    chan *bytes.Buffer
}

// NewBufferPool returns a newly allocated BufferPool with the given pool size
// and buffer size.
func NewBufferPool(size int, bufSize int) *BufferPool {
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
func (pool *BufferPool) Give(buf *bytes.Buffer) {
	select {
	case pool.pool <- buf:
	default:
	}
}
