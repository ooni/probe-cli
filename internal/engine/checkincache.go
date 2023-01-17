package engine

//
// Implements caching values passed by the check-in API.
//

import (
	"encoding/json"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// checkInFlagsState is the state created by check-in flags.
const checkInFlagsState = "checkinflags.state"

// checkInFlagsWrapper is the struct wrapping the check-in flags.
//
// See https://github.com/ooni/probe/issues/2396 for the reference issue
// describing adding feature flags to ooniprobe.
type checkInFlagsWrapper struct {
	// Expire contains the expiration date.
	Expire time.Time

	// Flags contains the actual flags.
	Flags map[string]bool
}

// updateCheckInFlagsState updates the state created by check-in flags.
func (s *Session) updateCheckInFlagsState(resp *model.OOAPICheckInResult) error {
	wrapper := &checkInFlagsWrapper{
		Expire: time.Now().Add(24 * time.Hour),
		Flags:  resp.Conf.Features,
	}
	data, err := json.Marshal(wrapper)
	runtimex.PanicOnError(err, "json.Marshal unexpectedly failed")
	return s.kvStore.Set(checkInFlagsState, data)
}

// getCheckInFlagValue returns the value of a check-in feature flag. In
// case of any error this function will return a false value.
func (s *Session) getCheckInFlagValue(name string) bool {
	data, err := s.kvStore.Get(checkInFlagsState)
	if err != nil {
		return false // as documented
	}
	var wrapper checkInFlagsWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return false // as documented
	}
	if time.Now().After(wrapper.Expire) {
		return false // as documented
	}
	return wrapper.Flags[name] // works even if map is nil
}
