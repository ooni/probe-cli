package webconnectivityqa

import (
	"encoding/json"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// testKeys is the test keys structure returned by this package.
type testKeys struct {
	// Accessible indicates whether the URL was accessible.
	Accessible optional.Value[bool] `json:"accessible"`

	// Blocking is either nil or a string classifying the blocking type.
	Blocking optional.Value[bool] `json:"blocking"`
}

// newTestKeys constructs the test keys from the measurement.
func newTestKeys(measurement *model.Measurement) *testKeys {
	rawTk := runtimex.Try1(json.Marshal(measurement.TestKeys))
	var tk testKeys
	runtimex.Try0(json.Unmarshal(rawTk, &tk))
	return &tk
}

// compareTestKeys compares the test keys by converting them to maps. This function
// return an explanatory error in case of mismatch and nil in case of match.
func compareTestKeys(left, right *testKeys) error {
	if d := cmp.Diff(testKeysToMap(left), testKeysToMap(right)); d != "" {
		return fmt.Errorf("test keys mismatch: %s", d)
	}
	return nil
}

func testKeysToMap(tk *testKeys) map[string]any {
	rawTk := runtimex.Try1(json.Marshal(tk))
	var mapTk map[string]any
	runtimex.Try0(json.Unmarshal(rawTk, &mapTk))
	return mapTk
}
