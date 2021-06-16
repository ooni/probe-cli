// Package torsf contains the experimental tor+snowflake test.
package torsf

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/ptx"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
)

// testVersion is the tor experiment version.
const testVersion = "0.1.0"

// Config contains the experiment config.
type Config struct{}

// TestKeys contains the experiment's result.
type TestKeys struct {
	// BootstrapTime contains the bootstrap time on success.
	BootstrapTime float64 `json:"bootstrap_time"`

	// Failure contains the failure or nil.
	Failure *string `json:"failure"`
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return "torsf"
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

func (m *Measurer) registerExtensions(measurement *model.Measurement) {
}

// Run implements model.ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	m.registerExtensions(measurement)
	testkeys := &TestKeys{}
	measurement.TestKeys = testkeys
	sfdialer := &ptx.SnowflakeDialer{}
	ptl := &ptx.Listener{
		PTDialer: sfdialer,
		Logger:   sess.Logger(),
	}
	if err := ptl.Start(); err != nil {
		// TODO(bassosimone): should we set any specific error
		// inside of the test keys in this case?
		return err
	}
	defer ptl.Stop()
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	tun, err := tunnel.Start(ctx, &tunnel.Config{
		Name:      "tor",
		Session:   sess,
		TunnelDir: "/tmp", // TODO(bassosimone): figure out
		Logger:    sess.Logger(),
		TorArgs: []string{
			"UseBridges", "1",
			"ClientTransportPlugin", ptl.AsClientTransportPluginArgument(),
			"Bridge", sfdialer.AsBridgeArgument(),
		},
	})
	if err != nil {
		testkeys.Failure = archival.NewFailure(err)
		return nil // this is basically a good test with a failure
	}
	defer tun.Stop()
	testkeys.BootstrapTime = tun.BootstrapTime().Seconds()
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
