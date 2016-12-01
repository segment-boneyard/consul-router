package main

import (
	"io"
	"sync"
)

// bufferPool is a simple wrapper around a sync.Pool that stores byte slices.
type bufferPool struct {
	pool sync.Pool
}

func makeBufferPool(size int) bufferPool {
	return bufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

func (p *bufferPool) get() []byte {
	return p.pool.Get().([]byte)
}

func (p *bufferPool) put(b []byte) {
	p.pool.Put(b)
}

// Copy bytes from w to r using a temporary buffer allocated from the global
// buffer pool.
func copyBytes(w io.Writer, r io.Reader) {
	b := buffers.get()
	io.CopyBuffer(w, r, b)
	buffers.put(b)
}

var (
	// A global buffer pool to be used for acquiring temporary buffers anywhere
	// in the program.
	buffers = makeBufferPool(16384)
)
