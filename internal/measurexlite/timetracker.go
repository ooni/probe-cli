package measurexlite

//
// Time tracking with facilities for deterministic testing.
//

import (
	"sync"
	"time"
)

// timeTracker tracks the evolution of time and allows for unit testing.
//
// Since
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
//
// Sub
//
// Likewise, in deterministic mode, each Sub operation returns a
// time increment equivalent to one second. However, Sub does not
// change the internal state of the time tracker.
type timeTracker struct {
	// counter is the counter used to return deterministic elapsed times.
	counter time.Duration

	// mu is a mutex protecting counter.
	mu sync.Mutex
}

// Since returns the elapsed time since a given zero time.
func (tt *timeTracker) Since(t0 time.Time) time.Duration {
	if tt != nil {
		return tt.next()
	}
	return time.Since(t0)
}

// next returns the next value of the internal counter. This
// function can safely be called by concurrent code.
func (tt *timeTracker) next() time.Duration {
	tt.mu.Lock()
	counter := tt.counter
	counter += time.Second
	tt.mu.Unlock()
	return counter
}

// Sub returns the difference of two points in time.
func (tt *timeTracker) Sub(t1, t0 time.Time) time.Duration {
	if tt != nil {
		return time.Second
	}
	return t1.Sub(t0)
}
