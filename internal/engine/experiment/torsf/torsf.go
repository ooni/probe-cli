// Package torsf contains the torsf experiment. This experiment
// measures the bootstrapping of tor using snowflake.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-030-torsf.md
package torsf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/ptx"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
)

// testVersion is the tor experiment version.
const testVersion = "0.2.0"

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

	// DefaultTimeout contains the default timeout for torsf
	DefaultTimeout float64 `json:"default_timeout"`

	// Failure contains the failure string or nil.
	Failure *string `json:"failure"`

	// PersistentDatadir indicates whether we're using a persistent tor datadir.
	PersistentDatadir bool `json:"persistent_datadir"`

	// RendezvousMethod contains the method used to perform the rendezvous.
	RendezvousMethod string `json:"rendezvous_method"`

	// TorLogs contains the bootstrap logs.
	TorLogs []string `json:"tor_logs"`

	// TorVersion contains the version of tor (if it's possible to obtain it).
	TorVersion string `json:"tor_version"`
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
	ptl, sfdialer, err := m.setup(ctx, sess.Logger())
	if err != nil {
		// we cannot setup the experiment
		return err
	}
	defer ptl.Stop()
	m.registerExtensions(measurement)
	start := time.Now()
	const maxRuntime = 600 * time.Second
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

// bootstrap runs the bootstrap.
func (m *Measurer) bootstrap(ctx context.Context, timeout time.Duration, sess model.ExperimentSession,
	out chan<- *TestKeys, ptl *ptx.Listener, sfdialer *ptx.SnowflakeDialer) {
	tk := &TestKeys{
		BootstrapTime:     0,
		DefaultTimeout:    timeout.Seconds(),
		Failure:           nil,
		PersistentDatadir: !m.config.DisablePersistentDatadir,
		RendezvousMethod:  sfdialer.RendezvousMethod.Name(),
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
		// Note: archival.NewFailure scrubs IP addresses
		tk.Failure = archival.NewFailure(err)
		return
	}
	defer tun.Stop()
	tk.BootstrapTime = tun.BootstrapTime().Seconds()
}

// torProgressRegexp helps to extract progress info from logs.
//
// See https://regex101.com/r/cer3lm/1.
var torProgressRegexp = regexp.MustCompile(
	`^[A-Za-z0-9.: ]+ \[notice\] Bootstrapped [0-9]+% \([A-Za-z_]+\): [A-Za-z0-9 ]+$`)

// readTorLogs attempts to read and include the tor logs into
// the test keys if this operation is possible.
//
// This function aims to _only_ include notice information about
// bootstrap according to the torProgressRegexp regexp.
//
// Tor is know to be good software that does not break its output
// unnecessarily and that does not include PII into its logs unless
// explicitly asked to. This fact gives me confidence that we can
// safely include this subset of the logs into the results.
//
// On this note, I think it's safe to include timestamps from the
// logs into the output, since we have a timestamp for the whole
// experiment already, so we don't leak much more by also including
// the Tor proper timestamps into the results.
func (m *Measurer) readTorLogs(logger model.Logger, tk *TestKeys, logFilePath string) {
	if logFilePath == "" {
		log.Warn("the tunnel claims there is no log file")
		return
	}
	data, err := os.ReadFile(logFilePath)
	if err != nil {
		log.Warnf("could not read tor logs: %s", err.Error())
		return
	}
	for _, bline := range bytes.Split(data, []byte("\n")) {
		if torProgressRegexp.Match(bline) {
			tk.TorLogs = append(tk.TorLogs, string(bline))
		}
	}
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
