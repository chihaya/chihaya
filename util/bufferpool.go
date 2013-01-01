/*
 * This file is part of Chihaya.
 *
 * Chihaya is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Chihaya is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Chihaya.  If not, see <http://www.gnu.org/licenses/>.
 */

package util

import (
	"bytes"
)

type BufferPool struct {
	bufSize int
	pool    chan *bytes.Buffer
}

func NewBufferPool(size int, bufSize int) *BufferPool {
	return &BufferPool{
		bufSize,
		make(chan *bytes.Buffer, size),
	}
}

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

func (pool *BufferPool) Give(buf *bytes.Buffer) {
	select {
	case pool.pool <- buf:
	default:
	}
}
