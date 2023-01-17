package dslx

import (
	"io"
	"sync"
)

// ConnPool tracks established connections.
type ConnPool struct {
	mu sync.Mutex
	v  []io.Closer
}

// MaybeTrack tracks the given connection if not nil. This
// method is safe for use by multiple goroutines.
func (p *ConnPool) MaybeTrack(c io.Closer) {
	if c != nil {
		defer p.mu.Unlock()
		p.mu.Lock()
		p.v = append(p.v, c)
	}
}

// Close closes all the tracked connections in reverse order. This
// method is safe for use by multiple goroutines.
func (p *ConnPool) Close() error {
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
