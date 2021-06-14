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
package atomicx

import "sync"

// Int64 is an int64 with atomic semantics.
type Int64 struct {
	// mu provides mutual exclusion.
	mu sync.Mutex

	// v is the underlying value.
	v int64
}

// Add behaves like atomic.AddInt64.
func (i64 *Int64) Add(delta int64) int64 {
	i64.mu.Lock()
	defer i64.mu.Unlock()
	i64.v += delta
	return i64.v
}

// Load behaves like atomic.LoadInt64.
func (i64 *Int64) Load() (v int64) {
	return i64.Add(0)
}
