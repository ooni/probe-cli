package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"time"
)

//
// Definition of the result of a network measurement.
//

const (
	// DefaultProbeIP is the default probe IP.
	DefaultProbeIP = "127.0.0.1"
)

// MeasurementTarget is the target of a OONI measurement.
type MeasurementTarget string

// MarshalJSON serializes the MeasurementTarget.
func (t MeasurementTarget) MarshalJSON() ([]byte, error) {
	if t == "" {
		return json.Marshal(nil)
	}
	return json.Marshal(string(t))
}

// Measurement is a OONI measurement.
//
// This structure is compatible with the definition of the base data format in
// https://github.com/ooni/spec/blob/master/data-formats/df-000-base.md.
type Measurement struct {
	// Annotations contains results annotations
	Annotations map[string]string `json:"annotations,omitempty"`

	// DataFormatVersion is the version of the data format
	DataFormatVersion string `json:"data_format_version"`

	// Extensions contains information about the extensions included
	// into the test_keys of this measurement.
	Extensions map[string]int64 `json:"extensions,omitempty"`

	// ID is the locally generated measurement ID
	ID string `json:"id,omitempty"`

	// Input is the measurement input
	Input MeasurementTarget `json:"input"`

	// InputHashes contains input hashes
	InputHashes []string `json:"input_hashes,omitempty"`

	// MeasurementStartTime is the time when the measurement started
	MeasurementStartTime string `json:"measurement_start_time"`

	// MeasurementStartTimeSaved is the moment in time when we
	// started the measurement. This is not included into the JSON
	// and is only used within probe-engine as a "zero" time.
	MeasurementStartTimeSaved time.Time `json:"-"`

	// Options contains command line options
	Options []string `json:"options,omitempty"`

	// ProbeASN contains the probe autonomous system number
	ProbeASN string `json:"probe_asn"`

	// ProbeCC contains the probe country code
	ProbeCC string `json:"probe_cc"`

	// ProbeCity contains the probe city
	ProbeCity string `json:"probe_city,omitempty"`

	// ProbeIP contains the probe IP
	ProbeIP string `json:"probe_ip,omitempty"`

	// ProbeNetworkName contains the probe network name
	ProbeNetworkName string `json:"probe_network_name"`

	// ReportID contains the report ID
	ReportID string `json:"report_id"`

	// ResolverASN is the ASN of the resolver
	ResolverASN string `json:"resolver_asn"`

	// ResolverIP is the resolver IP
	ResolverIP string `json:"resolver_ip"`

	// ResolverNetworkName is the network name of the resolver.
	ResolverNetworkName string `json:"resolver_network_name"`

	// SoftwareName contains the software name
	SoftwareName string `json:"software_name"`

	// SoftwareVersion contains the software version
	SoftwareVersion string `json:"software_version"`

	// TestHelpers contains the test helpers. It seems this structure is more
	// complex than we would like. In particular, using a map from string to
	// string does not fit into the web_connectivity use case. Hence, for now
	// we're going to represent this using interface{}. In going forward we
	// may probably want to have more uniform test helpers.
	TestHelpers map[string]interface{} `json:"test_helpers,omitempty"`

	// TestKeys contains the real test result. This field is opaque because
	// each experiment will insert here a different structure.
	TestKeys interface{} `json:"test_keys"`

	// TestName contains the test name
	TestName string `json:"test_name"`

	// MeasurementRuntime contains the measurement runtime. The JSON name
	// is test_runtime because this is the name expected by the OONI backend
	// even though that name is clearly a misleading one.
	MeasurementRuntime float64 `json:"test_runtime"`

	// TestStartTime contains the test start time
	TestStartTime string `json:"test_start_time"`

	// TestVersion contains the test version
	TestVersion string `json:"test_version"`
}

// AddAnnotations adds the annotations from input to m.Annotations.
func (m *Measurement) AddAnnotations(input map[string]string) {
	for key, value := range input {
		m.AddAnnotation(key, value)
	}
}

// AddAnnotation adds a single annotations to m.Annotations.
func (m *Measurement) AddAnnotation(key, value string) {
	if m.Annotations == nil {
		m.Annotations = make(map[string]string)
	}
	m.Annotations[key] = value
}

// ErrInvalidProbeIP indicates that we're dealing with a string that
// is not the valid serialization of an IP address.
var ErrInvalidProbeIP = errors.New("model: invalid probe IP")

// Scrub scrubs the probeIP out of the measurement.
func (m *Measurement) Scrub(probeIP string) (err error) {
	// We now behave like we can share everything except the
	// probe IP, which we instead cannot ever share
	m.ProbeIP = DefaultProbeIP
	return m.MaybeRewriteTestKeys(probeIP, json.Marshal)
}

// Scrubbed is the string that replaces IP addresses.
const Scrubbed = `[scrubbed]`

// MaybeRewriteTestKeys is the function called by Scrub that
// ensures that m's serialization doesn't include the IP
func (m *Measurement) MaybeRewriteTestKeys(
	currentIP string, marshal func(interface{}) ([]byte, error)) error {
	if net.ParseIP(currentIP) == nil {
		return ErrInvalidProbeIP
	}
	data, err := marshal(m.TestKeys)
	if err != nil {
		return err
	}
	// The check using Count is to save an unnecessary copy performed by
	// ReplaceAll when there are no matches into the body. This is what
	// we would like the common case to be, meaning that the code has done
	// its job correctly and has not leaked the IP.
	bpip := []byte(currentIP)
	if bytes.Count(data, bpip) <= 0 {
		return nil
	}
	data = bytes.ReplaceAll(data, bpip, []byte(Scrubbed))
	// We add an annotation such that hopefully later we can measure the
	// number of cases where we failed to sanitize properly.
	m.AddAnnotation("_probe_engine_sanitize_test_keys", "true")
	return json.Unmarshal(data, &m.TestKeys)
}
