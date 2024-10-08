package keywords

import (
	"sync"
)

// Pool is a type-safe wrapper around a sync.Pool.
type Pool struct {
	p *sync.Pool
}

// NewPool constructs a new Pool.
func NewPool(size int) *Pool {
	return &Pool{p: &sync.Pool{
		New: func() interface{} {
			return &Buffer{bs: make([]byte, 0, size)}
		},
	}}
}

// Get retrieves a Buffer from the pool, creating one if necessary.
func (p *Pool) Get() *Buffer {
	buf := p.p.Get().(*Buffer)
	buf.pool = p
	return buf
}
