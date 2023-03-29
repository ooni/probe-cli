package miniengine

//
// Measurement session
//

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/platform"
	"github.com/ooni/probe-cli/v3/internal/probeservices"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.org/x/net/context"
)

// SessionConfig contains configuration for a [Session]. The zero value is
// invalid; please, initialize all the fields marked as MANDATORY.
type SessionConfig struct {
	// BackendURL allows you to OPTIONALLY force the
	// usage of a specific OONI backend instance.
	BackendURL string `json:"backend_url"`

	// Proxy allows you to OPTIONALLY force a specific proxy
	// rather than using no proxy (the default).
	//
	// Use `psiphon:///` to force using Psiphon with the
	// embedded configuration file. Not all builds have
	// an embedded configuration file, but OONI builds have
	// such a file, so they can use this functionality.
	//
	// Use `tor:///` and `torsf:///` to respectively use vanilla tor
	// and tor plus snowflake as tunnels.
	//
	// Use `socks5://127.0.0.1:9050/` to connect to a SOCKS5
	// proxy running on 127.0.0.1:9050. This could be, for example,
	// a suitably configured `tor` instance.
	ProxyURL string `json:"proxy_url"`

	// SnowflakeRendezvousMethod OPTIONALLY allows you to specify
	// which snowflake rendezvous method to use. Valid methods to use
	// here are "amp" and "domain_fronting".
	SnowflakeRendezvousMethod string `json:"snowflake_rendezvous_method"`

	// SoftwareName is the MANDATORY name of the application
	// that will be using the new [Session].
	SoftwareName string `json:"software_name"`

	// SoftwareVersion is the MANDATORY version of the application
	// that will be using the new [Session].
	SoftwareVersion string `json:"software_version"`

	// StateDir is the MANDATORY directory where to store state
	// information required by a [Session].
	StateDir string `json:"state_dir"`

	// TempDir is the MANDATORY directory inside which the [Session] shall
	// store temporary files deleted when we close the [Session].
	TempDir string `json:"temp_dir"`

	// TorArgs contains OPTIONAL arguments to pass to the "tor" binary
	// when ProxyURL is `tor:///` or `torsf:///`.
	TorArgs []string `json:"tor_args"`

	// TorBinary is the OPTIONAL "tor" binary to use. When using this code
	// on mobile devices, we link with tor directly, so there is no need to
	// specify this argument when running on a mobile device.
	TorBinary string `json:"tor_binary"`

	// TunnelDir is the MANDATORY directory where the [Session] shall store
	// persistent data regarding circumvention tunnels.
	TunnelDir string `json:"tunnel_dir"`

	// Verbose OPTIONALLY configures the [Session] logger to be verbose.
	Verbose bool `json:"verbose"`
}

// ErrSessionConfig indicates that the [SessionConfig] is invalid.
var ErrSessionConfig = errors.New("invalid SessionConfig")

// check checks whether the [SessionConfig] is valid.
func (sc *SessionConfig) check() error {
	if sc.SoftwareName == "" {
		return fmt.Errorf("%w: %s", ErrSessionConfig, "SoftwareName is empty")
	}
	if sc.SoftwareVersion == "" {
		return fmt.Errorf("%w: %s", ErrSessionConfig, "SoftwareVersion is empty")
	}
	if sc.StateDir == "" {
		return fmt.Errorf("%w: %s", ErrSessionConfig, "StateDir is empty")
	}
	if sc.TempDir == "" {
		return fmt.Errorf("%w: %s", ErrSessionConfig, "TempDir is empty")
	}
	if sc.TunnelDir == "" {
		return fmt.Errorf("%w: %s", ErrSessionConfig, "TunnelDir is empty")
	}
	return nil
}

// mkdirAll ensures all the required directories exist.
func (sc *SessionConfig) mkdirAll() error {
	if err := os.MkdirAll(sc.StateDir, 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(sc.TempDir, 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(sc.TunnelDir, 0700); err != nil {
		return err
	}
	return nil
}

// Session is a measurement session. The zero value is invalid; please
// create a new instance using the [NewSession] factory.
type Session struct {
	// child is a thread-safe wrapper for the underlying [engine.Session].
	child *engine.Session

	// config is the [engine.SessionConfig] to use.
	config *engine.SessionConfig

	// closeJustOnce ensures we close this [Session] just once.
	closeJustOnce sync.Once

	// emitter is the [emitter] to use.
	emitter chan *Event

	// logger is the [model.Logger] to use.
	logger model.Logger

	// mu provides mutual exclusion.
	mu sync.Mutex

	// psc is the [probeservices.Client] to use.
	psc *probeservices.Client

	// submitter is the [probeservices.Submitter] to use.
	submitter *probeservices.Submitter
}

// NewSession creates a new [Session] instance.
func NewSession(config *SessionConfig) (*Session, error) {
	// check whether the [SessionConfig] is valid.
	if err := config.check(); err != nil {
		return nil, err
	}

	// make sure all the required directories exist.
	if err := config.mkdirAll(); err != nil {
		return nil, err
	}

	// create keyvalue store inside the user provided StateDir.
	kvstore, err := kvstore.NewFS(config.StateDir)
	if err != nil {
		return nil, err
	}

	// honor user-provided backend service, if any.
	var availableps []model.OOAPIService
	if config.BackendURL != "" {
		availableps = append(availableps, model.OOAPIService{
			Address: config.BackendURL,
			Type:    "https",
		})
	}

	// honor user-provided proxy, if any.
	var proxyURL *url.URL
	if config.ProxyURL != "" {
		var err error
		proxyURL, err = url.Parse(config.ProxyURL)
		if err != nil {
			return nil, err
		}
	}

	// create the base event emitter
	const buffer = 1024
	emitter := make(chan *Event, buffer)

	// create a logger using the base event emitter
	logger := newLoggerEmitter(emitter, config.Verbose)

	// create the underlying session using the [engine] package.
	engineConfig := engine.SessionConfig{
		AvailableProbeServices: availableps,
		KVStore:                kvstore,
		Logger:                 logger,
		ProxyURL:               proxyURL,
		SoftwareName:           config.SoftwareName,
		SoftwareVersion:        config.SoftwareVersion,
		TempDir:                config.TempDir,
		TorArgs:                config.TorArgs,
		TorBinary:              config.TorBinary,
		SnowflakeRendezvous:    config.SnowflakeRendezvousMethod,
		TunnelDir:              config.TunnelDir,
	}

	// assemble and return a session.
	sess := &Session{
		child:         nil,
		config:        &engineConfig,
		closeJustOnce: sync.Once{},
		emitter:       emitter,
		logger:        logger,
		mu:            sync.Mutex{},
	}
	return sess, nil
}

// Platform returns the operating system platform name.
func (sess *Session) Platform() string {
	return platform.Name()
}

// SoftwareName returns the configured software name.
func (sess *Session) SoftwareName() string {
	return sess.config.SoftwareName
}

// SoftwareVersion returns the configured software version.
func (sess *Session) SoftwareVersion() string {
	return sess.config.SoftwareVersion
}

// Bootstrap ensures that we bootstrap the [Session].
func (sess *Session) Bootstrap(ctx context.Context) *Task[Void] {
	task := &Task[Void]{
		done:    make(chan any),
		events:  sess.emitter,
		failure: nil,
		result:  Void{},
	}

	go func() {
		// synchronize with Task.Result
		defer close(task.done)

		// create a new underlying session instance
		child, err := engine.NewSession(ctx, *sess.config)
		if err != nil {
			task.failure = err
			return
		}

		// create a probeservices client
		psc, err := child.NewProbeServicesClient(ctx)
		if err != nil {
			child.Close()
			task.failure = err
			return
		}

		// create a submitter
		submitter, err := child.NewSubmitter(ctx)
		if err != nil {
			child.Close()
			task.failure = err
			return
		}

		// lock and store the underlying fields
		defer sess.mu.Unlock()
		sess.mu.Lock()
		sess.child = child
		sess.psc = psc
		sess.submitter = submitter
	}()

	return task
}

// ErrNoBootstrap indicates that you did not bootstrap the [Session].
var ErrNoBootstrap = errors.New("bootstrap the session first")

// CheckIn invokes the backend check-in API using the [Session].
func (sess *Session) CheckIn(
	ctx context.Context, config *model.OOAPICheckInConfig) *Task[*model.OOAPICheckInResult] {
	task := &Task[*model.OOAPICheckInResult]{
		done:    make(chan any),
		events:  sess.emitter,
		failure: nil,
		result:  nil,
	}

	go func() {
		// synchronize with Task.Result
		defer close(task.done)

		// lock and access the underlying session
		sess.mu.Lock()
		defer sess.mu.Unlock()

		// handle the case where we did not bootstrap
		if sess.child == nil {
			task.failure = ErrNoBootstrap
			return
		}
		runtimex.Assert(sess.psc != nil, "sess.psc is nil")

		// invoke the backend check-in API
		result, err := sess.psc.CheckIn(ctx, *config)
		if err != nil {
			task.failure = err
			return
		}

		// pass result to the caller
		task.result = result
	}()

	return task
}

// Geolocate uses the [Session] to geolocate the probe.
func (sess *Session) Geolocate(ctx context.Context) *Task[*Location] {
	task := &Task[*Location]{
		done:    make(chan any),
		events:  sess.emitter,
		failure: nil,
		result:  nil,
	}

	go func() {
		// synchronize with Task.Result
		defer close(task.done)

		// lock and access the underlying session
		sess.mu.Lock()
		defer sess.mu.Unlock()

		// handle the case where we did not bootstrap
		if sess.child == nil {
			task.failure = ErrNoBootstrap
			return
		}

		// perform geolocation and handle failure
		if err := sess.child.MaybeLookupLocationContext(ctx); err != nil {
			task.failure = err
			return
		}

		// copy results to the caller
		task.result = &Location{
			ProbeASN:            int64(sess.child.ProbeASN()),
			ProbeASNString:      sess.child.ProbeASNString(),
			ProbeCC:             sess.child.ProbeCC(),
			ProbeNetworkName:    sess.child.ProbeNetworkName(),
			ProbeIP:             sess.child.ProbeIP(),
			ResolverASN:         int64(sess.child.ResolverASN()),
			ResolverASNString:   sess.child.ResolverASNString(),
			ResolverIP:          sess.child.ResolverIP(),
			ResolverNetworkName: sess.child.ResolverNetworkName(),
		}
	}()

	return task
}

// MeasurementResult contains the results of [Session.Measure]
type MeasurementResult struct {
	// KibiBytesReceived contains the KiB we received
	KibiBytesReceived float64

	// KibiBytesSent contains the KiB we sent
	KibiBytesSent float64

	// Measurement is the generated [model.Measurement]
	Measurement *model.Measurement `json:"measurement"`

	// Summary is the corresponding summary.
	Summary any `json:"summary"`
}

// Measure performs a measurement using the given experiment name, the
// given input, and the given opaque experiment options.
func (sess *Session) Measure(
	ctx context.Context,
	name string,
	input string,
	options map[string]any,
) *Task[*MeasurementResult] {
	task := &Task[*MeasurementResult]{
		done:    make(chan any),
		events:  sess.emitter,
		failure: nil,
		result:  nil,
	}

	go func() {
		// synchronize with Task.Result
		defer close(task.done)

		// lock and access the underlying session
		sess.mu.Lock()
		defer sess.mu.Unlock()

		// handle the case where we did not bootstrap
		if sess.child == nil {
			task.failure = ErrNoBootstrap
			return
		}

		// TODO(bassosimone): there is a bug where we create a new report ID for
		// each measurement because there's a different TestStartTime

		// create a [model.ExperimentBuilder]
		builder, err := sess.child.NewExperimentBuilder(name)
		if err != nil {
			task.failure = err
			return
		}

		// set the proper callbacks for the experiment
		callbacks := &callbacks{sess.emitter}
		builder.SetCallbacks(callbacks)

		// set the proper options for the experiment
		builder.SetOptionsAny(options)

		// create an experiment instance
		exp := builder.NewExperiment()

		// perform the measurement
		meas, err := exp.MeasureWithContext(ctx, input)
		if err != nil {
			task.failure = err
			return
		}

		// obtain the summary
		summary, err := exp.GetSummaryKeys(meas)
		if err != nil {
			task.failure = err
			return
		}

		// pass response to the caller
		task.result = &MeasurementResult{
			KibiBytesReceived: exp.KibiBytesReceived(),
			KibiBytesSent:     exp.KibiBytesSent(),
			Measurement:       meas,
			Summary:           summary,
		}

	}()

	return task
}

// Submit submits a [model.Measurement] to the OONI backend.
func (sess *Session) Submit(ctx context.Context, meas *model.Measurement) *Task[string] {
	task := &Task[string]{
		done:    make(chan any),
		events:  sess.emitter,
		failure: nil,
		result:  "",
	}

	go func() {
		// synchronize with Task.Result
		defer close(task.done)

		// lock and access the underlying session
		sess.mu.Lock()
		defer sess.mu.Unlock()

		// handle the case where we did not bootstrap
		if sess.child == nil {
			task.failure = ErrNoBootstrap
			return
		}
		runtimex.Assert(sess.submitter != nil, "sess.psc is nil")

		// submit without causing data races
		reportID, err := sess.submitter.SubmitWithoutModifyingMeasurement(ctx, meas)
		if err != nil {
			task.failure = err
			return
		}

		// pass the reportID to the caller
		task.result = reportID
	}()

	return task
}

// Close closes the [Session]. This function does not attempt
// to close an already closed [Session].
func (sess *Session) Close() (err error) {
	sess.closeJustOnce.Do(func() {
		sess.mu.Lock()
		if sess.child != nil {
			err = sess.child.Close()
			sess.child = nil
		}
		sess.mu.Unlock()
	})
	return err
}
