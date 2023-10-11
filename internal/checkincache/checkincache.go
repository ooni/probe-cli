// Package checkincache contains an on-disk cache for check-in responses.
package checkincache

import (
	"encoding/json"
	"fmt"
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

// Store stores the result of the latest check-in in the given key-value store.
//
// We store check-in feature flags in a file called checkinflags.state. These flags
// are valid for 24 hours, after which we consider them stale.
func Store(kvStore model.KeyValueStore, resp *model.OOAPICheckInResult) error {
	// store the check-in flags in the key-value store
	wrapper := &checkInFlagsWrapper{
		Expire: time.Now().Add(24 * time.Hour),
		Flags:  resp.Conf.Features,
	}
	data, err := json.Marshal(wrapper)
	runtimex.PanicOnError(err, "json.Marshal unexpectedly failed")
	return kvStore.Set(checkInFlagsState, data)
}

// GetFeatureFlag returns the value of a check-in feature flag. In case of any
// error this function will always return a false value.
func GetFeatureFlag(kvStore model.KeyValueStore, name string) bool {
	data, err := kvStore.Get(checkInFlagsState)
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

// ExperimentEnabledKey returns the [model.KeyValueStore] key to use to
// know whether a disabled experiment has been enabled via check-in.
func ExperimentEnabledKey(name string) string {
	return fmt.Sprintf("%s_enabled", name)
}

// ExperimentEnabled returns whether a given experiment has been enabled by a previous
// execution of check-in. Some experiments are disabled by default for different reasons
// and we use the check-in API to control whether and when they should be enabled.
func ExperimentEnabled(kvStore model.KeyValueStore, name string) bool {
	return GetFeatureFlag(kvStore, ExperimentEnabledKey(name))
}
