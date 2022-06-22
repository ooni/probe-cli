package measurexlite

//
// Time tracking with facilities for deterministic testing.
//

import (
	"sync"
	"time"
)

// timeTracker tracks the evolution of time.
//
// The nil structure calls functions in the stdlib's time package
// such as time.Since. Normal code should always use a nil timeTracker
// struct. We use non-nil TimeTrackers for testing.
//
// A non-nil structure returns deterministic timing: each Since
// call increments an internal counter and returns to the caller
// the previous value of the counter. So, you are able to get
// deterministic time readings inside unit tests. Each invocation
// of Since in deterministic mode increments the counter by 1 second.
type timeTracker struct {
	// counter is the counter used to return deterministic elapsed times.
	counter time.Duration

	// mu is a mutex protecting counter.
	mu sync.Mutex
}

// Since returns the elapsed time since a given zero time.
//
// If the tt pointer is nil, this function is equivalent to calling
// time.Since. Otherwise, we return a deterministic duration as
// documented in timeTracker's documentation.
func (tt *timeTracker) Since(t0 time.Time) time.Duration {
	if tt != nil {
		tt.mu.Lock()
		counter := tt.counter
		counter += time.Second
		tt.mu.Unlock()
		return counter
	}
	return time.Since(t0)
}
