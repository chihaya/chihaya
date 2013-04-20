// This file is part of Chihaya.
//
// Chihaya is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Chihaya is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Chihaya.  If not, see <http://www.gnu.org/licenses/>.

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
