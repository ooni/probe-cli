package archival

//
// Saver implementation
//

import (
	"sync"
)

// Saver allows to save network, DNS, QUIC, TLS, HTTP events.
//
// You MUST use NewSaver to create a new instance.
type Saver struct {
	// mu provides mutual exclusion.
	mu sync.Mutex

	// trace is the current trace.
	trace *Trace
}

// NewSaver creates a new Saver instance.
//
// You MUST use this function to create a Saver.
func NewSaver() *Saver {
	return &Saver{
		mu:    sync.Mutex{},
		trace: &Trace{},
	}
}

// MoveOutTrace moves the current trace out of the saver and
// creates a new empty trace inside it.
func (as *Saver) MoveOutTrace() *Trace {
	as.mu.Lock()
	t := as.trace
	as.trace = &Trace{}
	as.mu.Unlock()
	return t
}
