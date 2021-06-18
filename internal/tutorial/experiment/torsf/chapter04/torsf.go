package main

import (
	"context"
	"path"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/ptx"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
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
	sfdialer := &ptx.SnowflakeDialer{}
	ptl := &ptx.Listener{
		PTDialer: sfdialer,
		Logger:   sess.Logger(),
	}
	if err := ptl.Start(); err != nil {
		testkeys.Failure = archival.NewFailure(err)
		// This error condition mostly means "I could not open a local
		// listening port", which strikes as fundamental failure.
		errch <- err
		return
	}
	defer ptl.Stop()
	tun, err := tunnel.Start(ctx, &tunnel.Config{
		Name:      "tor",
		Session:   sess,
		TunnelDir: path.Join(sess.TempDir(), "torsf"),
		Logger:    sess.Logger(),
		TorArgs: []string{
			"UseBridges", "1",
			"ClientTransportPlugin", ptl.AsClientTransportPluginArgument(),
			"Bridge", sfdialer.AsBridgeArgument(),
		},
	})
	if err != nil {
		// Note: archival.NewFailure scrubs IP addresses
		testkeys.Failure = archival.NewFailure(err)
		// This error condition means we could not bootstrap with snowflake
		// for $reasons, so the experiment didn't fail, rather it did record
		// that something prevented snowflake from running.
		errch <- nil
		return
	}
	defer tun.Stop()
	testkeys.BootstrapTime = tun.BootstrapTime().Seconds()
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
