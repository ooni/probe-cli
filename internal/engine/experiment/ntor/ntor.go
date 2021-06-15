// Package ntor contains the new implementation of the tor experiment.
//
// This package will eventually replace the tor package.
//
// Spec: https://github.com/ooni/spec/blob/master/nettests/ts-023-tor.md.
package ntor

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// testVersion is the tor experiment version.
const testVersion = "0.4.0"

// Config contains the experiment config.
type Config struct{}

// TestKeys contains the experiment's result.
type TestKeys struct{}

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

// Run implements model.ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	testkeys := &TestKeys{}
	measurement.TestKeys = testkeys
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
