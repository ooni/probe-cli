// Package vanillator contains the vanilla_tor experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-016-vanilla-tor.md
package vanillator

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/torlogs"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
)

// Implementation note: this file is written with easy diffing with respect
// to internal/engine/experiment/torsf/torsf.go in mind.
//
// We may want to have a single implementation for both nettests in the future.

// testVersion is the experiment version.
const testVersion = "0.2.0"

// Config contains the experiment config.
type Config struct {
	// DisableProgress disables printing progress messages.
	DisableProgress bool `ooni:"Disable printing progress messages"`
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

	// TransportName is always set to "vanilla" for this experiment.
	TransportName string `json:"transport_name"`
}

// Measurer performs the measurement.
type Measurer struct {
	// config contains the experiment settings.
	config Config

	// mockStartTunnel is an optional function that allows us to override the
	// default tunnel.Start function used to start a tunnel.
	mockStartTunnel func(
		ctx context.Context, config *tunnel.Config) (tunnel.Tunnel, tunnel.DebugInfo, error)
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return "vanilla_tor"
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
const maxRuntime = 200 * time.Second

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
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, maxRuntime)
	defer cancel()
	tkch := make(chan *TestKeys)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	go m.bootstrap(ctx, maxRuntime, sess, tkch)
	for {
		select {
		case tk := <-tkch:
			measurement.TestKeys = tk
			callbacks.OnProgress(1.0, "vanilla_tor experiment is finished")
			return nil
		case <-ticker.C:
			if !m.config.DisableProgress {
				elapsedTime := time.Since(start)
				progress := elapsedTime.Seconds() / maxRuntime.Seconds()
				callbacks.OnProgress(progress, fmt.Sprintf(
					"vanilla_tor: elapsedTime: %.0f s; maxRuntime: %.0f s",
					elapsedTime.Seconds(), maxRuntime.Seconds()))
			}
		}
	}
}

// values for the backward compatible error field.
var (
	timeoutReachedError = "timeout-reached"
	unknownError        = "unknown-error"
)

// bootstrap runs the bootstrap.
func (m *Measurer) bootstrap(ctx context.Context, timeout time.Duration,
	sess model.ExperimentSession, out chan<- *TestKeys) {
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
		Timeout:       timeout.Seconds(),
		TransportName: "vanilla",
	}
	defer func() {
		out <- tk
	}()
	tun, debugInfo, err := m.startTunnel()(ctx, &tunnel.Config{
		Name:      "tor",
		Session:   sess,
		TunnelDir: path.Join(m.baseTunnelDir(sess), "vanillator"),
		Logger:    sess.Logger(),
	})
	tk.TorVersion = debugInfo.Version
	m.readTorLogs(sess.Logger(), tk, debugInfo.LogFilePath)
	if err != nil {
		// Note: archival.NewFailure scrubs IP addresses
		tk.Failure = archival.NewFailure(err)
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
	return sess.TempDir()
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
	errInvalidTestKeysType = errors.New("vanilla_tor: invalid test keys type")

	//errNilTestKeys indicates that the test keys are nil.
	errNilTestKeys = errors.New("vanilla_tor: nil test keys")
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
