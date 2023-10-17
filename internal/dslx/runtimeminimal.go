package dslx

import (
	"io"
	"sync"
)

// NewMinimalRuntime creates a minimal [Runtime] implementation.
func NewMinimalRuntime() *MinimalRuntime {
	return &MinimalRuntime{
		mu: sync.Mutex{},
		v:  []io.Closer{},
	}
}

// MinimalRuntime is a minimal [Runtime] implementation.
type MinimalRuntime struct {
	mu sync.Mutex
	v  []io.Closer
}

// MaybeTrackConn implements Runtime.
func (p *MinimalRuntime) MaybeTrackConn(conn io.Closer) {
	if conn != nil {
		defer p.mu.Unlock()
		p.mu.Lock()
		p.v = append(p.v, conn)
	}
}

// Close implements Runtime.
func (p *MinimalRuntime) Close() error {
	// Implementation note: reverse order is such that we close TLS
	// connections before we close the TCP connections they use. Hence
	// we'll _gracefully_ close TLS connections.
	defer p.mu.Unlock()
	p.mu.Lock()
	for idx := len(p.v) - 1; idx >= 0; idx-- {
		_ = p.v[idx].Close()
	}
	p.v = nil // reset
	return nil
}
