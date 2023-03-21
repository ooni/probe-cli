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

/*
	Design note
	-----------

	We cannot store on disk the monotonic clock readings because we are
	only allowed to serialize time.Time as a string representing a "wall
	clock" time and we cannot serialize the underlying, and system
	dependent, monotonic clock reading. Additionally, on Windows, the
	monotonic clock uses the interrupt time, which countes the time since
	boot in 100ns intervals, so serializing it is dangerous.

	See https://cs.opensource.google/go/go/+/refs/tags/go1.20.2:src/runtime/time_windows_amd64.s
	See https://learn.microsoft.com/en-us/windows/win32/sysinfo/interrupt-time
	See https://learn.microsoft.com/en-us/windows/win32/sysinfo/system-time

	Storing wall clocks on disk could be done in UTC to account for the
	user jumping across timezones. Still, even if we did that, the stored
	reading would still be wrong when there are wall clock adjusments, which
	includes adjusting the system clock using NTP et al. (There are cases
	in which the clock adjustment is a jump and cases in which it is relatively
	monotonic; see https://man.openbsd.org/settimeofday#CAVEATS.)

	Therefore, the implementation of this package works entirely on memory
	and requires calling the check-in API before performing any other
	operation that requires timing.

	This design choice is ~fine since we're moving towards a direction
	where the check-in API will be called before running tests. Yet,
	it also means that we wouldn't be able to verify the probe clock when
	using a cached check-in response in the future.

	A possible future improvement would be for this package to export a
	"best effort" clock using either the system clock or the API clock
	to provide time.Time values to TLS and QUIC. The choice on which
	clock to use would depend on the measured clocks offset. This would
	theoretically allow us to make TLS handshakes work even when the
	probe clock is singificantly off. However, it's also true that we
	would like to make TLS handshake verification non-fatal, because it
	helps us to collect more data. Because of this reason, I have not
	implemented subverting the TLS verification reference clock.
*/

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
	if !s.good {
		return
	}
	const smallOffset = 5 * time.Minute
	shouldWarn := s.offset < -smallOffset || s.offset > smallOffset
	if shouldWarn {
		logger.Warnf("checkintime: the probe clock is off by %s", s.offset)
	}
}
