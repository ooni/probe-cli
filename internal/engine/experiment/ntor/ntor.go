// Package ntor contains the new implementation of the tor experiment.
//
// This package will eventually replace the tor package.
//
// Spec: https://github.com/ooni/spec/blob/master/nettests/ts-023-tor.md.
package ntor

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
)

// testVersion is the tor experiment version.
const testVersion = "0.4.0"

// Config contains the experiment config.
type Config struct{}

// TestKeys contains the experiment's result.
type TestKeys struct {
	// Targets maps each target name to its measurement results.
	Targets map[string]TargetResults `json:"targets"`
}

// TargetResults contains the results of measuring a target.
type TargetResults struct {
	// Failure is the failure that occurred or nil.
	Failure *string `json:"failure"`

	// NetworkEvents contains the network events.
	NetworkEvents []archival.NetworkEvent `json:"network_events"`

	// Requests contains the HTTP requests.
	Requests []archival.RequestEntry `json:"requests"`

	// TargetAddress is the target's address.
	TargetAddress string `json:"target_address"`

	// TargetName is the target's name.
	TargetName string `json:"target_name,omitempty"`

	// TargetProtocol is the target's protocol.
	TargetProtocol string `json:"target_protocol"`

	// TargetSource is the source from which we obtained the target.
	TargetSource string `json:"target_source,omitempty"`

	// TCPConnect contains the TCP connect events.
	TCPConnect []archival.TCPConnectEntry `json:"tcp_connect"`

	// TLSHandshakes contains the TLS handshake events.
	TLSHandshakes []archival.TLSHandshake `json:"tls_handshakes"`
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m Measurer) ExperimentName() string {
	return "tor"
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

func (m *Measurer) registerExtensions(measurement *model.Measurement) {
	archival.ExtHTTP.AddTo(measurement)
	archival.ExtNetevents.AddTo(measurement)
	archival.ExtDNS.AddTo(measurement)
	archival.ExtTCPConnect.AddTo(measurement)
	archival.ExtTLSHandshake.AddTo(measurement)
}

// Run implements model.ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	m.registerExtensions(measurement)
	testkeys := &TestKeys{}
	measurement.TestKeys = testkeys
	targets, err := m.fetchTargets(ctx, sess)
	if err != nil {
		// TODO(bassosimone): should we set any specific error
		// inside of the test keys in this case?
		return err
	}
	testkeys.Targets = m.measure(ctx, sess.Logger(), targets)
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
