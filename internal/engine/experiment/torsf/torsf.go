// Package torsf contains the torsf experiment. This experiment
// measures the bootstrapping of tor using snowflake.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-030-torsf.md
package torsf

import (
	"context"
	"path"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/ptx"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
)

// testVersion is the tor experiment version.
const testVersion = "0.1.1"

// Config contains the experiment config.
type Config struct {
	DisableProgress bool `ooni:"Disable printing progress messages"`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	// BootstrapTime contains the bootstrap time on success.
	BootstrapTime float64 `json:"bootstrap_time"`

	// Failure contains the failure string or nil.
	Failure *string `json:"failure"`
}

// Measurer performs the measurement.
type Measurer struct {
	// config contains the experiment settings.
	config Config

	// mockStartListener is an optional function that allows us to override
	// the function we actually use to start the ptx listener.
	mockStartListener func() error

	// mockStartTunnel is an optional function that allows us to override the
	// default tunnel.Start function used to start a tunnel.
	mockStartTunnel func(ctx context.Context, config *tunnel.Config) (tunnel.Tunnel, error)
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return "torsf"
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

// registerExtensions registers the extensions used by this experiment.
func (m *Measurer) registerExtensions(measurement *model.Measurement) {
	// currently none
}

// Run runs the experiment with the specified context, session,
// measurement, and experiment calbacks. This method should only
// return an error in case the experiment could not run (e.g.,
// a required input is missing). Otherwise, the code should just
// set the relevant OONI error inside of the measurement and
// return nil. This is important because the caller may not submit
// the measurement if this method returns an error.
func (m *Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	m.registerExtensions(measurement)
	testkeys := &TestKeys{}
	measurement.TestKeys = testkeys
	start := time.Now()
	const maxRuntime = 600 * time.Second
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
			if !m.config.DisableProgress {
				progress := time.Since(start).Seconds() / maxRuntime.Seconds()
				callbacks.OnProgress(progress, "torsf experiment is running")
			}
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
	if err := m.startListener(ptl.Start); err != nil {
		testkeys.Failure = archival.NewFailure(err)
		// This error condition mostly means "I could not open a local
		// listening port", which strikes as fundamental failure.
		errch <- err
		return
	}
	defer ptl.Stop()
	tun, err := m.startTunnel()(ctx, &tunnel.Config{
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

// startListener either calls f or mockStartListener depending
// on whether mockStartListener is nil or not.
func (m *Measurer) startListener(f func() error) error {
	if m.mockStartListener != nil {
		return m.mockStartListener()
	}
	return f()
}

// startTunnel returns the proper function to start a tunnel.
func (m *Measurer) startTunnel() func(
	ctx context.Context, config *tunnel.Config) (tunnel.Tunnel, error) {
	if m.mockStartTunnel != nil {
		return m.mockStartTunnel
	}
	return tunnel.Start
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
func (m *Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
