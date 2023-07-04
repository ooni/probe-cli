package miniengine

//
// The "bootstrap" task
//

import (
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/platform"
	"github.com/ooni/probe-cli/v3/internal/probeservices"
	"golang.org/x/net/context"
)

// BootstrapConfig contains the config for [Session.Bootstrap]. The zero value
// is invalid; please fill all fields marked as MANDATORY.
type BootstrapConfig struct {
	// BackendURL allows you to OPTIONALLY force the
	// usage of a specific OONI backend instance.
	BackendURL string `json:"backend_url"`

	// CategoryCodes contains OPTIONAL category codes for the check-in API.
	CategoryCodes []string `json:"category_codes"`

	// Charging is the OPTIONAL charging hint for the check-in API.
	Charging bool `json:"charging"`

	// OnWiFi is the OPTIONAL on-wifi hint for the check-in API.
	OnWiFi bool `json:"on_wifi"`

	// ProxyURL allows you to OPTIONALLY force a specific proxy
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

	// RunType is the MANDATORY run-type for the check-in API.
	RunType model.RunType `json:"run_type"`

	// SnowflakeRendezvousMethod OPTIONALLY allows you to specify
	// which snowflake rendezvous method to use. Valid methods to use
	// here are "amp" and "domain_fronting".
	SnowflakeRendezvousMethod string `json:"snowflake_rendezvous_method"`

	// TorArgs contains OPTIONAL arguments to pass to the "tor" binary
	// when ProxyURL is `tor:///` or `torsf:///`.
	TorArgs []string `json:"tor_args"`

	// TorBinary is the OPTIONAL "tor" binary to use. When using this code
	// on mobile devices, we link with tor directly, so there is no need to
	// specify this argument when running on a mobile device.
	TorBinary string `json:"tor_binary"`
}

// TODO(bassosimone): rather than having calls that return the geolocation and
// the result of the check-in, we should modify Bootstrap to return something
// like a Task[BootstrapResult] that contains both. The Bootstrap will still be
// idempotent and short circuit already existing results if they are available.
//
// By doing that, we would simplify the corresponding C API.

// Bootstrap ensures that we bootstrap the [Session]. This function
// is safe to call multiple times. We'll only bootstrap on the first
// invocation and do nothing for subsequent invocations.
func (s *Session) Bootstrap(ctx context.Context, config *BootstrapConfig) *Task[Void] {
	task := &Task[Void]{
		done:    make(chan any),
		events:  s.emitter,
		failure: nil,
		result:  Void{},
	}
	go s.bootstrapAsync(ctx, config, task)
	return task
}

// bootstrapAsync runs the bootstrap in a background goroutine.
func (s *Session) bootstrapAsync(ctx context.Context, config *BootstrapConfig, task *Task[Void]) {
	// synchronize with Task.Result
	defer close(task.done)

	// make the whole operation locked with respect to s
	defer s.mu.Unlock()
	s.mu.Lock()

	// handle the case where bootstrap already occurred while we were locked
	if !s.state.IsNone() {
		return
	}

	// perform a sync bootstrap
	err := s.bootstrapSyncLocked(ctx, config)

	// pass result to the caller
	task.failure = err
}

// bootstrapSyncLocked executes a synchronous bootstrap. This function MUST be
// run while holding the s.mu mutex because it mutates s.
func (s *Session) bootstrapSyncLocked(ctx context.Context, config *BootstrapConfig) error {
	// create a new instance of the [engineSessionState] type.
	ess, err := s.newEngineSessionState(ctx, config)
	if err != nil {
		return err
	}

	// MUTATE s to store the state
	s.state = optional.Some(ess)
	return nil
}

// newEngineSessionState creates a new instance of [engineSessionState].
func (s *Session) newEngineSessionState(
	ctx context.Context, config *BootstrapConfig) (*engineSessionState, error) {
	// create configuration for [engine.NewSession]
	engineConfig, err := s.newEngineSessionConfig(config)
	if err != nil {
		return nil, err
	}

	// create a new underlying session instance
	child, err := engine.NewSession(ctx, *engineConfig)
	if err != nil {
		return nil, err
	}

	// create a probeservices client
	psc, err := child.NewProbeServicesClient(ctx)
	if err != nil {
		child.Close()
		return nil, err
	}

	// geolocate the probe.
	location, err := s.geolocate(ctx, child)
	if err != nil {
		child.Close()
		return nil, err
	}

	// lookup the available backends.
	if err := child.MaybeLookupBackendsContext(ctx); err != nil {
		child.Close()
		return nil, err
	}

	// call the check-in API.
	resp, err := s.checkIn(ctx, location, child, psc, config)
	if err != nil {
		child.Close()
		return nil, err
	}

	// create [engineSessionState]
	ess := &engineSessionState{
		checkIn: resp,
		geoloc:  location,
		psc:     psc,
		sess:    child,
	}
	return ess, nil
}

// geolocate performs the geolocation during the bootstrap.
func (s *Session) geolocate(ctx context.Context, sess *engine.Session) (*Location, error) {
	// perform geolocation and handle failure
	if err := sess.MaybeLookupLocationContext(ctx); err != nil {
		return nil, err
	}

	// copy the result of the geolocation
	location := &Location{
		ProbeASN:            int64(sess.ProbeASN()),
		ProbeASNString:      sess.ProbeASNString(),
		ProbeCC:             sess.ProbeCC(),
		ProbeNetworkName:    sess.ProbeNetworkName(),
		ProbeIP:             sess.ProbeIP(),
		ResolverASN:         int64(sess.ResolverASN()),
		ResolverASNString:   sess.ResolverASNString(),
		ResolverIP:          sess.ResolverIP(),
		ResolverNetworkName: sess.ResolverNetworkName(),
	}
	return location, nil
}

// checkIn invokes the checkIn API.
func (s *Session) checkIn(
	ctx context.Context,
	location *Location,
	sess *engine.Session,
	psc *probeservices.Client,
	config *BootstrapConfig,
) (*model.OOAPICheckInResult, error) {
	categoryCodes := config.CategoryCodes
	if len(categoryCodes) <= 0 {
		// make sure it not nil because this would
		// actually break the check-in API
		categoryCodes = []string{}
	}
	apiConfig := model.OOAPICheckInConfig{
		Charging:        config.Charging,
		OnWiFi:          config.OnWiFi,
		Platform:        platform.Name(),
		ProbeASN:        location.ProbeASNString,
		ProbeCC:         location.ProbeCC,
		RunType:         config.RunType,
		SoftwareName:    sess.SoftwareName(),
		SoftwareVersion: sess.SoftwareVersion(),
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: categoryCodes,
		},
	}
	return psc.CheckIn(ctx, apiConfig)
}

// newEngineSessionConfig creates a new [engine.SessionConfig] instance.
func (s *Session) newEngineSessionConfig(config *BootstrapConfig) (*engine.SessionConfig, error) {
	// create keyvalue store inside the user provided stateDir.
	kvstore, err := kvstore.NewFS(s.stateDir)
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

	// create the underlying session using the [engine] package.
	engineConfig := &engine.SessionConfig{
		AvailableProbeServices: availableps,
		KVStore:                kvstore,
		Logger:                 s.logger,
		ProxyURL:               proxyURL,
		SoftwareName:           s.softwareName,
		SoftwareVersion:        s.softwareVersion,
		TempDir:                s.tempDir,
		TorArgs:                config.TorArgs,
		TorBinary:              config.TorBinary,
		SnowflakeRendezvous:    config.SnowflakeRendezvousMethod,
		TunnelDir:              s.tunnelDir,
	}

	return engineConfig, nil
}
