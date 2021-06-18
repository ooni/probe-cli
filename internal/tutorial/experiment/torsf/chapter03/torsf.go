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

// TestKeys contains the experiment results.
type TestKeys struct {
	// BootstrapTime is the time required to bootstrap.
	BootstrapTime float64 `json:"bootstrap_time"`

	// Failure is the failure that occurred, or nil.
	Failure *string `json:"failure"`
}

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	testkeys := &TestKeys{}
	measurement.TestKeys = testkeys
	start := time.Now()
	const maxRuntime = 300 * time.Second
	ctx, cancel := context.WithTimeout(ctx, maxRuntime)
	defer cancel()
	errch := make(chan error)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	go m.run(ctx, sess, testkeys, errch)
	for {
		select {
		case err := <-errch:
			callbacks.OnProgress(1.0, "torsf experiment is finished")
			return err
		case <-ticker.C:
			progress := time.Since(start).Seconds() / maxRuntime.Seconds()
			callbacks.OnProgress(progress, "torsf experiment is running")
		}
	}
}

// run runs the bootstrap. This function ONLY returns an error when
// there has been a fundamental error starting the test. This behavior
// follows the expectations for the ExperimentMeasurer.Run method.
func (m *Measurer) run(ctx context.Context,
	sess model.ExperimentSession, testkeys *TestKeys, errch chan<- error) {
	// TODO: implement the bootstrap
	fakeBootstrapTime := 10 * time.Second
	time.Sleep(fakeBootstrapTime)
	testkeys.BootstrapTime = fakeBootstrapTime.Seconds()
	errch <- nil
}

// SummaryKeys contains summary keys for this experiment.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m *Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return &SummaryKeys{IsAnomaly: false}, nil
}
