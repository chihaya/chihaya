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
