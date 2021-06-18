package main

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// Config contains config for the torsf experiment.
type Config struct{}

// Measurer is the torsf measurer.
type Measurer struct {
	config Config
}

// newExperimentMeasurer creates a new ExperimentMeasurer for torsf.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// ExperimentName implements ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return "torsf"
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return "0.1.0"
}

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	// TODO: implement the real experiment
	time.Sleep(time.Second)
	sess.Logger().Info("hello from the torsf experiment!")
	return nil
}

// SummaryKeys contains summary keys for this experiment.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m *Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return &SummaryKeys{IsAnomaly: false}, nil
}
