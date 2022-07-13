package testingx

import (
	"sync"
	"time"
)

// TimeDeterministic implements time.Now in a deterministic fashion
// such that every time.Time call returns a moment in time that occurs
// one second after the configured zeroTime.
//
// It's safe to use this struct from multiple goroutine contexts.
type TimeDeterministic struct {
	// counter counts the number of "ticks" passed since the zero time: each
	// call to Now increments this counter by one second.
	counter time.Duration

	// mu protects fields in this structure from concurrent access.
	mu sync.Mutex

	// zeroTime is the lazy-initialized zero time. The first call to Now
	// will initialize this field with the current time.
	zeroTime time.Time
}

// NewTimeDeterministic creates a new instance using the given zeroTime value.
func NewTimeDeterministic(zeroTime time.Time) *TimeDeterministic {
	return &TimeDeterministic{
		counter:  0,
		mu:       sync.Mutex{},
		zeroTime: zeroTime,
	}
}

// Now is like time.Now but more deterministic. The first call returns the
// configured zeroTime and subsequent calls return moments in time that occur
// exactly one second after the time returned by the previous call.
func (td *TimeDeterministic) Now() time.Time {
	td.mu.Lock()
	if td.zeroTime.IsZero() {
		td.zeroTime = time.Now()
	}
	offset := td.counter
	td.counter += time.Second
	res := td.zeroTime.Add(offset)
	td.mu.Unlock()
	return res
}
