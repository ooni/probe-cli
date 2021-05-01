// Package atomicx extends sync/atomic.
//
// Sync/atomic fails when using int64 atomic operations on 32 bit platforms
// when the access is not aligned. As specified in the documentation, in
// fact, "it is the caller's responsibility to arrange for 64-bit alignment
// of 64-bit words accessed atomically". For more information on this
// issue, see https://golang.org/pkg/sync/atomic/#pkg-note-BUG.
//
// As explained in CONTRIBUTING.md, probe-cli SHOULD use this package rather
// than sync/atomic to avoid these alignment issues on 32 bit.
//
// It is of course possible to write atomic code using 64 bit variables on a
// 32 bit platform, but that's difficult to do correctly. This package
// provides an easier-to-use interface. We use allocated
// structures protected by a mutex that encapsulate a int64 value.
//
// While there we also added support for atomic float64 operations, again
// by using structures protected by a mutex variable.
package atomicx

import "sync"

// Int64 is an int64 with atomic semantics.
type Int64 struct {
	mu sync.Mutex
	v  int64
}

// NewInt64 creates a new int64 with atomic semantics.
func NewInt64() *Int64 {
	return new(Int64)
}

// Add behaves like atomic.AddInt64
func (i64 *Int64) Add(delta int64) (newvalue int64) {
	i64.mu.Lock()
	i64.v += delta
	newvalue = i64.v
	i64.mu.Unlock()
	return
}

// Load behaves like atomic.LoadInt64
func (i64 *Int64) Load() (v int64) {
	i64.mu.Lock()
	v = i64.v
	i64.mu.Unlock()
	return
}

// Float64 is an float64 with atomic semantics.
type Float64 struct {
	mu sync.Mutex
	v  float64
}

// NewFloat64 creates a new float64 with atomic semantics.
func NewFloat64() *Float64 {
	return new(Float64)
}

// Add behaves like AtomicInt64.Add but for float64
func (f64 *Float64) Add(delta float64) (newvalue float64) {
	f64.mu.Lock()
	f64.v += delta
	newvalue = f64.v
	f64.mu.Unlock()
	return
}

// Load behaves like LoadInt64.Load buf for float64
func (f64 *Float64) Load() (v float64) {
	f64.mu.Lock()
	v = f64.v
	f64.mu.Unlock()
	return
}
