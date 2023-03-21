// Package checkintime tracks the check-in API time. By tracking such
// a time we can perform the following actions:
//
// 1. submit measurements with a reference time based on the check-in API
// time rather than on the probe clock;
//
// 2. warn the user that the probe clock is definitely off.
//
// See https://github.com/ooni/probe/issues/1781 for more details.
package checkintime

import (
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// state contains the [checkintime] state. The zero value
// of this structure is ready to use.
type state struct {
	// apiTime contains the time according to the check-in API.
	apiTime time.Time

	// good indicates whether we have good data.
	good bool

	// monotonicTimeUTC contains the monotonic UTC clock reading when we
	// saved the apiTime. We need this variable because times unmarshalled
	// from JSON contain no monotonic clock readings.
	//
	// See https://github.com/golang/go/blob/72c58fb/src/time/time.go#L58.
	monotonicTimeUTC time.Time

	// offset is the offset between monotonicTimeUTC and apiTime.
	offset time.Duration

	// mu provides mutual exclusion.
	mu sync.Mutex
}

// singleton is the [checkintime] singleton.
var singleton = &state{}

// Save saves the time according to the check-in API.
func Save(cur time.Time) {
	singleton.save(cur)
}

func (s *state) save(cur time.Time) {
	if cur.IsZero() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apiTime = cur
	s.good = true
	s.monotonicTimeUTC = time.Now().UTC()
	s.offset = s.monotonicTimeUTC.Sub(s.apiTime) // UNRELIABLE non-monotonic diff
}

// Now returns the current time using as zero time the time saved by
// [Save] rather than the system clock. The time returned by this call
// is reliable because it consists of the diff between two monotonic
// clock readings of the system's monotonic clock added to the zero time
// reference provided by the check-in API. When the probe clock is OK,
// this value will always be slightly in the past, because we cannot
// account for the time elapsed transferring the check-in response.
func Now() (time.Time, bool) {
	return singleton.now()
}

func (s *state) now() (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.good {
		return time.Time{}, false
	}
	delta := time.Since(s.monotonicTimeUTC) // RELIABLE diff of monotonic readings
	out := s.apiTime.Add(delta)
	return out, true
}

// MaybeWarnAboutProbeClockBeingOff emits a warning if the probe clock is off
// compared to the clock used by the check-in API.
func MaybeWarnAboutProbeClockBeingOff(logger model.Logger) {
	singleton.maybeWarnAboutProbeClockBeingOff(logger)
}

func (s *state) maybeWarnAboutProbeClockBeingOff(logger model.Logger) {
	const smallOffset = 5 * time.Minute
	shouldWarn := s.offset < -smallOffset || s.offset > smallOffset
	if shouldWarn {
		logger.Warnf("checkintime: the probe clock is off by %s", s.offset)
	}
}
