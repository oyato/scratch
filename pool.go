package scratch

import (
	"sync"
)

// Pool is a wrapper around sync.Pool, holding Buf objects.
type Pool struct {
	p *sync.Pool
}

// Get return a buffer from the pool.
func (p *Pool) Get() *Buf {
	return p.p.Get().(*Buf)
}

// Put puts the buffer b into into the pool after resetting it.
func (p *Pool) Put(b *Buf) {
	if b == nil {
		return
	}
	b.Reset()
	p.p.Put(b)
}

// NewPool returns a new pool of buffers initially sized with capacity bufCap.
func NewPool(bufCap int) *Pool {
	return &Pool{p: &sync.Pool{
		New: func() interface{} {
			return NewBuf(bufCap)
		},
	}}
}
