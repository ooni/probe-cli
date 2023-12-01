package webconnectivityqa

import (
	"encoding/json"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// testKeys is the test keys structure returned by this package.
type testKeys struct {
	// XExperimentVersion is the experiment version.
	XExperimentVersion string `json:"x_experiment_version"`

	// DNSExperimentFailure contains the failure occurre during the DNS experiment.
	DNSExperimentFailure any `json:"dns_experiment_failure"`

	// DNSConsistency is either "consistent" or "inconsistent" and indicates whether the IP addresses
	// returned by the probe match those returned by the TH. When the probe DNS lookup fails and the
	// TH lookup succeeds (or the other way around) the DNSConsistency should be "inconsistent".
	DNSConsistency any `json:"dns_consistency"`

	// ControlFailure indicates whether the control connection failed.
	ControlFailure any `json:"control_failure"`

	// HTTPExperimentFailure indicates whether the HTTP experiment failed.
	HTTPExperimentFailure any `json:"http_experiment_failure"`

	// These keys indicate whether the HTTP body returned by the TH matches the probe's body.
	BodyLengthMatch any     `json:"body_length_match"`
	BodyProportion  float64 `json:"body_proportion"`
	StatusCodeMatch any     `json:"status_code_match"`
	HeadersMatch    any     `json:"headers_match"`
	TitleMatch      any     `json:"title_match"`

	// XStatus summarizes the result of the analysis performed by WebConnectivity v0.4.
	XStatus int64 `json:"x_status"`

	// These flags summarize the result of the analysis performed by WebConnectivity LTE.
	XDNSFlags      int64 `json:"x_dns_flags"`
	XBlockingFlags int64 `json:"x_blocking_flags"`
	XNullNullFlags int64 `json:"x_null_null_flags"`

	// Accessible indicates whether the URL was accessible.
	Accessible any `json:"accessible"`

	// Blocking is either nil or a string classifying the blocking type.
	Blocking any `json:"blocking"`
}

// newTestKeys constructs the test keys from the measurement.
func newTestKeys(measurement *model.Measurement) *testKeys {
	rawTk := runtimex.Try1(json.Marshal(measurement.TestKeys))
	var tk testKeys
	runtimex.Try0(json.Unmarshal(rawTk, &tk))
	tk.XExperimentVersion = measurement.TestVersion
	return &tk
}

// compareTestKeys compares two testKeys instances. It returns an error in
// case of a mismatch and returns nil otherwise.
func compareTestKeys(expected, got *testKeys) error {
	// always ignore the experiment version because it is not set inside the expected value
	options := []cmp.Option{
		cmpopts.IgnoreFields(testKeys{}, "XExperimentVersion"),
	}

	switch got.XExperimentVersion {
	case "0.4.3":
		// ignore the fields that are specific to LTE
		options = append(options, cmpopts.IgnoreFields(testKeys{}, "XDNSFlags", "XBlockingFlags", "XNullNullFlags"))

	case "0.5.26":
		// ignore the fields that are specific to v0.4
		options = append(options, cmpopts.IgnoreFields(testKeys{}, "XStatus"))

		// TODO(bassosimone): these flags are specific of the "orig" analysis engine
		options = append(options, cmpopts.IgnoreFields(testKeys{}, "XDNSFlags", "XBlockingFlags", "XNullNullFlags"))

	default:
		return fmt.Errorf("unknown experiment version: %s", got.XExperimentVersion)
	}

	// return an error if the comparison indicates there are differences
	if d := cmp.Diff(expected, got, options...); d != "" {
		return fmt.Errorf("test keys mismatch: %s", d)
	}
	return nil
}
