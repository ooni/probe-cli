package oonimkall

import "sync"

//
// This file contains mocks for types used by tasks. Because
// we only use mocks when testing, this file is a `_test.go` file.
//

// CollectorTaskEmitter is a thread-safe taskEmitter
// that stores all the events inside itself.
type CollectorTaskEmitter struct {
	// events contains the events
	events []*event

	// mu provides mutual exclusion
	mu sync.Mutex
}

// ensures that a CollectorTaskEmitter is a taskEmitter.
var _ taskEmitter = &CollectorTaskEmitter{}

// Emit implements the taskEmitter.Emit method.
func (e *CollectorTaskEmitter) Emit(key string, value interface{}) {
	e.mu.Lock()
	e.events = append(e.events, &event{Key: key, Value: value})
	e.mu.Unlock()
}

// Collect returns a copy of the collected events. It is safe
// to read the events. It's a data race to modify them.
//
// After this function has been called, the internal array
// of events will now be empty.
func (e *CollectorTaskEmitter) Collect() (out []*event) {
	e.mu.Lock()
	out = e.events
	e.events = nil
	e.mu.Unlock()
	return
}
