package checkintime

import (
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

// Make sure that we can compute times relative to the base time specified
// by the check-in API as opposed to the system clock. It does not matter
// which clock is wrong in this test, by the way. In reality, the wrong clock
// is the probe clock, while in this test the API clock is wrong.
func TestWorkingAsIntended(t *testing.T) {

	// This test covers the case where we've not initialized the state yet
	t.Run("when we have not set the apiTime yet", func(t *testing.T) {
		s := &state{}

		// we expect the current time to be unavailable
		t.Run("state.now", func(t *testing.T) {
			out, good := s.now()
			if good {
				t.Fatal("expected false here")
			}
			if !out.IsZero() {
				t.Fatal("expected zero value here")
			}
		})

		// we expect no warning here
		t.Run("state.maybeWarnAboutTheProbeClockBeingOff", func(t *testing.T) {
			var called bool
			logger := &mocks.Logger{
				MockWarnf: func(format string, v ...interface{}) {
					called = true
				},
			}
			s.maybeWarnAboutProbeClockBeingOff(logger)
			if called {
				t.Fatal("expected false here")
			}
		})
	})

	// This test covers the case where the input time is zero. It should not
	// happen in practice, because the parsing of the check-in API response
	// fails if the response does not contain a correctly formatted time string.
	t.Run("when the apiTime is zero", func(t *testing.T) {
		s := &state{}
		s.save(time.Time{}) // zero

		// we expect the current time to be unavailable
		t.Run("state.now", func(t *testing.T) {
			out, good := s.now()
			if good {
				t.Fatal("expected false here")
			}
			if !out.IsZero() {
				t.Fatal("expected zero value here")
			}
		})

		// we expect no warning here
		t.Run("state.maybeWarnAboutProbeClockBeingOff", func(t *testing.T) {
			var called bool
			logger := &mocks.Logger{
				MockWarnf: func(format string, v ...interface{}) {
					called = true
				},
			}
			s.maybeWarnAboutProbeClockBeingOff(logger)
			if called {
				t.Fatal("expected false here")
			}
		})
	})

	// This test covers the case where we've been given a valid value from
	// the check-in API, so we can compute offsets etc.
	t.Run("after we have set the apiTime", func(t *testing.T) {
		// create empty state
		s := &state{}

		// pretend the API time is some time in the past
		apiTime := time.Date(2022, 12, 23, 7, 36, 0, 0, time.UTC)
		s.save(apiTime)

		// await a little bit
		time.Sleep(time.Second)

		// obtain the current time according to [state]
		t.Run("state.now", func(t *testing.T) {
			now, good := s.now()

			// the current time must be good
			if !good {
				t.Fatal("expected to see true here")
			}

			// compute delta between now and the apiTime
			delta := now.Sub(apiTime)

			// make sure the elapsed time is around one second
			if delta < 700*time.Millisecond || delta > 1300*time.Millisecond {
				t.Fatal("expected around one second, got", delta.Seconds(), "seconds")
			}
		})

		// we expect a warning here
		t.Run("state.maybeWarnAboutProbeClockBeingOff", func(t *testing.T) {
			var called bool
			logger := &mocks.Logger{
				MockWarnf: func(format string, v ...interface{}) {
					called = true
				},
			}
			s.maybeWarnAboutProbeClockBeingOff(logger)
			if !called {
				t.Fatal("expected true here")
			}
		})
	})

	t.Run("additional tests to cover the public API", func(t *testing.T) {
		// save the current time as the API time
		apiTime := time.Now()
		Save(apiTime)

		// await a little bit
		time.Sleep(time.Second)

		// we expect to be able to get the current time
		t.Run("Now", func(t *testing.T) {
			now, good := Now()
			if !good {
				t.Fatal("expected to see true here")
			}
			delta := now.Sub(apiTime)
			if delta < 700*time.Millisecond || delta > 1300*time.Millisecond {
				t.Fatal("expected around one second, got", delta.Seconds(), "seconds")
			}
		})

		// we should not warn
		t.Run("MaybeWarnAboutProbeClockBeingOff", func(t *testing.T) {
			var called bool
			logger := &mocks.Logger{
				MockWarnf: func(format string, v ...interface{}) {
					called = true
				},
			}
			MaybeWarnAboutProbeClockBeingOff(logger)
			if called {
				t.Fatal("expected false here")
			}
		})
	})
}
