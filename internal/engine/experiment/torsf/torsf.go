// Package torsf contains the torsf experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-030-torsf.md
package torsf

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/ptx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/torlogs"
	"github.com/ooni/probe-cli/v3/internal/tracex"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
)

// Implementation note: this file is written with easy diffing with respect
// to internal/engine/experiment/vanillator/vanillator.go in mind.
//
// We may want to have a single implementation for both nettests in the future.

// testVersion is the experiment version.
const testVersion = "0.3.0"

// Config contains the experiment config.
type Config struct {
	// DisablePersistentDatadir disables using a persistent datadir.
	DisablePersistentDatadir bool `ooni:"Disable using a persistent tor datadir"`

	// DisableProgress disables printing progress messages.
	DisableProgress bool `ooni:"Disable printing progress messages"`

	// RendezvousMethod allows to choose the method with which to rendezvous.
	RendezvousMethod string `ooni:"Choose the method with which to rendezvous. Must be one of amp and domain_fronting. Leaving this field empty means we should use the default."`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	// BootstrapTime contains the bootstrap time on success.
	BootstrapTime float64 `json:"bootstrap_time"`

	// Error is one of `null`, `"timeout-reached"`, and `"unknown-error"` (this
	// field exists for backward compatibility with the previous
	// `vanilla_tor` implementation).
	Error *string `json:"error"`

	// Failure contains the failure string or nil.
	Failure *string `json:"failure"`

	// Success indicates whether we succeded (this field exists for
	// backward compatibility with the previous `vanilla_tor` implementation).
	Success bool `json:"success"`

	// PersistentDatadir indicates whether we're using a persistent tor datadir.
	PersistentDatadir bool `json:"persistent_datadir"`

	// RendezvousMethod contains the method used to perform the rendezvous.
	RendezvousMethod string `json:"rendezvous_method"`

	// Timeout contains the default timeout for this experiment
	Timeout float64 `json:"timeout"`

	// TorLogs contains the bootstrap logs.
	TorLogs []string `json:"tor_logs"`

	// TorProgress contains the percentage of the maximum progress reached.
	TorProgress int64 `json:"tor_progress"`

	// TorProgressTag contains the tag of the maximum progress reached.
	TorProgressTag string `json:"tor_progress_tag"`

	// TorProgressSummary contains the summary of the maximum progress reached.
	TorProgressSummary string `json:"tor_progress_summary"`

	// TorVersion contains the version of tor (if it's possible to obtain it).
	TorVersion string `json:"tor_version"`

	// TransportName is always set to "snowflake" for this experiment.
	TransportName string `json:"transport_name"`
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
	mockStartTunnel func(
		ctx context.Context, config *tunnel.Config) (tunnel.Tunnel, tunnel.DebugInfo, error)
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

// maxRuntime is the maximum runtime for this experiment
const maxRuntime = 600 * time.Second

// Run runs the experiment with the specified context, session,
// measurement, and experiment calbacks. This method should only
// return an error in case the experiment could not run (e.g.,
// a required input is missing). Otherwise, the code should just
// set the relevant OONI error inside of the measurement and
// return nil. This is important because the caller may not submit
// the measurement if this method returns an error.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session
	ptl, sfdialer, err := m.setup(ctx, sess.Logger())
	if err != nil {
		// we cannot setup the experiment
		return err
	}
	defer ptl.Stop()
	m.registerExtensions(measurement)
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, maxRuntime)
	defer cancel()
	tkch := make(chan *TestKeys)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	go m.bootstrap(ctx, maxRuntime, sess, tkch, ptl, sfdialer)
	for {
		select {
		case tk := <-tkch:
			measurement.TestKeys = tk
			callbacks.OnProgress(1.0, "torsf experiment is finished")
			return nil
		case <-ticker.C:
			if !m.config.DisableProgress {
				elapsedTime := time.Since(start)
				progress := elapsedTime.Seconds() / maxRuntime.Seconds()
				callbacks.OnProgress(progress, fmt.Sprintf(
					"torsf: elapsedTime: %.0f s; maxRuntime: %.0f s",
					elapsedTime.Seconds(), maxRuntime.Seconds()))
			}
		}
	}
}

// setup prepares for running the torsf experiment. Returns a valid ptx listener
// and snowflake dialer on success. Returns an error on failure. On success,
// remember to Stop the ptx listener when you're done.
func (m *Measurer) setup(ctx context.Context,
	logger model.Logger) (*ptx.Listener, *ptx.SnowflakeDialer, error) {
	rm, err := ptx.NewSnowflakeRendezvousMethod(m.config.RendezvousMethod)
	if err != nil {
		// cannot run the experiment with unknown rendezvous method
		return nil, nil, err
	}
	sfdialer := ptx.NewSnowflakeDialerWithRendezvousMethod(rm)
	ptl := &ptx.Listener{
		ExperimentByteCounter: bytecounter.ContextExperimentByteCounter(ctx),
		Logger:                logger,
		PTDialer:              sfdialer,
		SessionByteCounter:    bytecounter.ContextSessionByteCounter(ctx),
	}
	if err := m.startListener(ptl.Start); err != nil {
		// This error condition mostly means "I could not open a local
		// listening port", which strikes as fundamental failure.
		return nil, nil, err
	}
	logger.Infof("torsf: rendezvous method: '%s'", m.config.RendezvousMethod)
	return ptl, sfdialer, nil
}

// values for the backward compatible error field.
var (
	timeoutReachedError = "timeout-reached"
	unknownError        = "unknown-error"
)

// bootstrap runs the bootstrap.
func (m *Measurer) bootstrap(ctx context.Context, timeout time.Duration, sess model.ExperimentSession,
	out chan<- *TestKeys, ptl *ptx.Listener, sfdialer *ptx.SnowflakeDialer) {
	tk := &TestKeys{
		// initialized later
		BootstrapTime:      0,
		Error:              nil,
		Failure:            nil,
		Success:            false,
		TorLogs:            []string{},
		TorProgress:        0,
		TorProgressTag:     "",
		TorProgressSummary: "",
		TorVersion:         "",
		// initialized now
		PersistentDatadir: !m.config.DisablePersistentDatadir,
		RendezvousMethod:  sfdialer.RendezvousMethod.Name(),
		//
		Timeout:       timeout.Seconds(),
		TransportName: "snowflake",
	}
	sess.Logger().Infof(
		"torsf: disable persistent datadir: %+v", m.config.DisablePersistentDatadir)
	defer func() {
		out <- tk
	}()
	tun, debugInfo, err := m.startTunnel()(ctx, &tunnel.Config{
		Name:      "tor",
		Session:   sess,
		TunnelDir: path.Join(m.baseTunnelDir(sess), "torsf"),
		Logger:    sess.Logger(),
		TorArgs: []string{
			"UseBridges", "1",
			"ClientTransportPlugin", ptl.AsClientTransportPluginArgument(),
			"Bridge", sfdialer.AsBridgeArgument(),
		},
	})
	tk.TorVersion = debugInfo.Version
	m.readTorLogs(sess.Logger(), tk, debugInfo.LogFilePath)
	if err != nil {
		// Note: tracex.NewFailure scrubs IP addresses
		tk.Failure = tracex.NewFailure(err)
		if errors.Is(err, context.DeadlineExceeded) {
			tk.Error = &timeoutReachedError
		} else {
			tk.Error = &unknownError
		}
		tk.Success = false
		return
	}
	defer tun.Stop()
	tk.BootstrapTime = tun.BootstrapTime().Seconds()
	tk.Success = true
}

// readTorLogs attempts to read and include the tor logs into
// the test keys if this operation is possible.
func (m *Measurer) readTorLogs(logger model.Logger, tk *TestKeys, logFilePath string) {
	tk.TorLogs = append(tk.TorLogs, torlogs.ReadBootstrapLogsOrWarn(logger, logFilePath)...)
	if len(tk.TorLogs) <= 0 {
		return
	}
	last := tk.TorLogs[len(tk.TorLogs)-1]
	bi, err := torlogs.ParseBootstrapLogLine(last)
	// Implementation note: parsing cannot fail here because we're using the same code
	// for selecting and for parsing the bootstrap logs, so we panic on error.
	runtimex.PanicOnError(err, fmt.Sprintf("cannot parse bootstrap line: %s", last))
	tk.TorProgress = bi.Progress
	tk.TorProgressTag = bi.Tag
	tk.TorProgressSummary = bi.Summary
}

// baseTunnelDir returns the base directory to use for tunnelling
func (m *Measurer) baseTunnelDir(sess model.ExperimentSession) string {
	if m.config.DisablePersistentDatadir {
		return sess.TempDir()
	}
	return sess.TunnelDir()
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
	ctx context.Context, config *tunnel.Config) (tunnel.Tunnel, tunnel.DebugInfo, error) {
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
// Note that this structure is part of the ABI contract with ooniprobe
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

var (
	// errInvalidTestKeysType indicates the test keys type is invalid.
	errInvalidTestKeysType = errors.New("torsf: invalid test keys type")

	//errNilTestKeys indicates that the test keys are nil.
	errNilTestKeys = errors.New("torsf: nil test keys")
)

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m *Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	testkeys, good := measurement.TestKeys.(*TestKeys)
	if !good {
		return nil, errInvalidTestKeysType
	}
	if testkeys == nil {
		return nil, errNilTestKeys
	}
	return SummaryKeys{IsAnomaly: testkeys.Failure != nil}, nil
}
