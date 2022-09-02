package engine

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/sessionresolver"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/platform"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// SessionConfig contains the Session config
type SessionConfig struct {
	AvailableProbeServices []model.OOAPIService
	KVStore                model.KeyValueStore
	Logger                 model.Logger
	ProxyURL               *url.URL
	SoftwareName           string
	SoftwareVersion        string
	TempDir                string
	TorArgs                []string
	TorBinary              string

	// SnowflakeRendezvous is the rendezvous method
	// to be used by the torsf tunnel
	SnowflakeRendezvous string

	// TunnelDir is the directory where we should store
	// the state of persistent tunnels. This field is
	// optional _unless_ you want to use tunnels. In such
	// case, starting a tunnel will fail because there
	// is no directory where to store state.
	TunnelDir string
}

// Session is a measurement session. It contains shared information
// required to run a measurement session, and it controls the lifecycle
// of such resources. It is not possible to reuse a Session. You MUST
// NOT attempt to use a Session again after Session.Close.
type Session struct {
	availableProbeServices   []model.OOAPIService
	availableTestHelpers     map[string][]model.OOAPIService
	byteCounter              *bytecounter.Counter
	httpDefaultTransport     model.HTTPTransport
	kvStore                  model.KeyValueStore
	location                 *geolocate.Results
	logger                   model.Logger
	proxyURL                 *url.URL
	queryProbeServicesCount  *atomicx.Int64
	resolver                 *sessionresolver.Resolver
	selectedProbeServiceHook func(*model.OOAPIService)
	selectedProbeService     *model.OOAPIService
	softwareName             string
	softwareVersion          string
	tempDir                  string

	// closeOnce allows us to call Close just once.
	closeOnce sync.Once

	// mu provides mutual exclusion.
	mu sync.Mutex

	// testLookupLocationContext is a an optional hook for testing
	// allowing us to mock LookupLocationContext.
	testLookupLocationContext func(ctx context.Context) (*geolocate.Results, error)

	// testMaybeLookupBackendsContext is an optional hook for testing
	// allowing us to mock MaybeLookupBackendsContext.
	testMaybeLookupBackendsContext func(ctx context.Context) error

	// testMaybeLookupLocationContext is an optional hook for testing
	// allowing us to mock MaybeLookupLocationContext.
	testMaybeLookupLocationContext func(ctx context.Context) error

	// testNewProbeServicesClientForCheckIn is an optional hook for testing
	// allowing us to mock NewProbeServicesClient when calling CheckIn.
	testNewProbeServicesClientForCheckIn func(ctx context.Context) (
		sessionProbeServicesClientForCheckIn, error)

	// torArgs contains the optional arguments for tor that we may need
	// to pass to urlgetter when it uses a tor tunnel.
	torArgs []string

	// torBinary contains the optional path to the tor binary that we
	// may need to pass to urlgetter when it uses a tor tunnel.
	torBinary string

	// tunnelDir is the directory used by tunnels.
	tunnelDir string

	// tunnel is the optional tunnel that we may be using. It is created
	// by NewSession and it is cleaned up by Close.
	tunnel tunnel.Tunnel
}

// sessionProbeServicesClientForCheckIn returns the probe services
// client that we should be using for performing the check-in.
type sessionProbeServicesClientForCheckIn interface {
	CheckIn(ctx context.Context, config model.OOAPICheckInConfig) (*model.OOAPICheckInInfo, error)
}

// NewSession creates a new session. This factory function will
// execute the following steps:
//
// 1. Make sure the config is sane, apply reasonable defaults
// where possible, otherwise return an error.
//
// 2. Create a temporary directory.
//
// 3. Create an instance of the session.
//
// 4. If the user requested for a proxy that entails a tunnel (at the
// moment of writing this note, either psiphon or tor), then start the
// requested tunnel and configure it as our proxy.
//
// 5. Create a compound resolver for the session that will attempt
// to use a bunch of DoT/DoH servers before falling back to the system
// resolver if nothing else works (see the sessionresolver pkg). This
// sessionresolver will be using the configured proxy, if any.
//
// 6. Create the default HTTP transport that we should be using when
// we communicate with the OONI backends. This transport will be
// using the configured proxy, if any.
//
// If any of these steps fails, then we cannot create a measurement
// session and we return an error.
func NewSession(ctx context.Context, config SessionConfig) (*Session, error) {
	if config.Logger == nil {
		return nil, errors.New("Logger is empty")
	}
	if config.SoftwareName == "" {
		return nil, errors.New("SoftwareName is empty")
	}
	if config.SoftwareVersion == "" {
		return nil, errors.New("SoftwareVersion is empty")
	}
	if config.KVStore == nil {
		config.KVStore = &kvstore.Memory{}
	}
	// Implementation note: if config.TempDir is empty, then Go will
	// use the temporary directory on the current system. This should
	// work on Desktop. We tested that it did also work on iOS, but
	// we have also seen on 2020-06-10 that it does not work on Android.
	tempDir, err := ioutil.TempDir(config.TempDir, "ooniengine")
	if err != nil {
		return nil, err
	}
	sess := &Session{
		availableProbeServices:  config.AvailableProbeServices,
		byteCounter:             bytecounter.New(),
		kvStore:                 config.KVStore,
		logger:                  config.Logger,
		queryProbeServicesCount: &atomicx.Int64{},
		softwareName:            config.SoftwareName,
		softwareVersion:         config.SoftwareVersion,
		tempDir:                 tempDir,
		torArgs:                 config.TorArgs,
		torBinary:               config.TorBinary,
		tunnelDir:               config.TunnelDir,
	}
	proxyURL := config.ProxyURL
	if proxyURL != nil {
		switch proxyURL.Scheme {
		case "psiphon", "tor", "torsf", "fake":
			config.Logger.Infof(
				"starting '%s' tunnel; please be patient...", proxyURL.Scheme)
			tunnel, _, err := tunnel.Start(ctx, &tunnel.Config{
				Logger:              config.Logger,
				Name:                proxyURL.Scheme,
				SnowflakeRendezvous: config.SnowflakeRendezvous,
				Session:             &sessionTunnelEarlySession{},
				TorArgs:             config.TorArgs,
				TorBinary:           config.TorBinary,
				TunnelDir:           config.TunnelDir,
			})
			if err != nil {
				return nil, err
			}
			config.Logger.Infof("tunnel '%s' running...", proxyURL.Scheme)
			sess.tunnel = tunnel
			proxyURL = tunnel.SOCKS5ProxyURL()
		}
	}
	sess.proxyURL = proxyURL
	sess.resolver = &sessionresolver.Resolver{
		ByteCounter: sess.byteCounter,
		KVStore:     config.KVStore,
		Logger:      sess.logger,
		ProxyURL:    proxyURL,
	}
	txp := netxlite.NewHTTPTransportWithLoggerResolverAndOptionalProxyURL(
		sess.logger, sess.resolver, sess.proxyURL,
	)
	txp = bytecounter.WrapHTTPTransport(txp, sess.byteCounter)
	sess.httpDefaultTransport = txp
	return sess, nil
}

// TunnelDir returns the persistent directory used by tunnels.
func (s *Session) TunnelDir() string {
	return s.tunnelDir
}

// KibiBytesReceived accounts for the KibiBytes received by the HTTP clients
// managed by this session so far, including experiments.
func (s *Session) KibiBytesReceived() float64 {
	return s.byteCounter.KibiBytesReceived()
}

// KibiBytesSent is like KibiBytesReceived but for the bytes sent.
func (s *Session) KibiBytesSent() float64 {
	return s.byteCounter.KibiBytesSent()
}

// CheckIn calls the check-in API. The input arguments MUST NOT
// be nil. Before querying the API, this function will ensure
// that the config structure does not contain any field that
// SHOULD be initialized and is not initialized. Whenever there
// is a field that is not initialized, we will attempt to set
// a reasonable default value for such a field. This list describes
// the current defaults we'll choose:
//
// - Platform: if empty, set to Session.Platform();
//
// - ProbeASN: if empty, set to Session.ProbeASNString();
//
// - ProbeCC: if empty, set to Session.ProbeCC();
//
// - RunType: if empty, set to model.RunTypeTimed;
//
// - SoftwareName: if empty, set to Session.SoftwareName();
//
// - SoftwareVersion: if empty, set to Session.SoftwareVersion();
//
// - WebConnectivity.CategoryCodes: if nil, we will allocate
// an empty array (the API does not like nil).
//
// Because we MAY need to know the current ASN and CC, this
// function MAY call MaybeLookupLocationContext.
//
// The return value is either the check-in response or an error.
func (s *Session) CheckIn(
	ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInInfo, error) {
	if err := s.maybeLookupLocationContext(ctx); err != nil {
		return nil, err
	}
	client, err := s.newProbeServicesClientForCheckIn(ctx)
	if err != nil {
		return nil, err
	}
	if config.Platform == "" {
		config.Platform = s.Platform()
	}
	if config.ProbeASN == "" {
		config.ProbeASN = s.ProbeASNString()
	}
	if config.ProbeCC == "" {
		config.ProbeCC = s.ProbeCC()
	}
	if config.RunType == "" {
		config.RunType = model.RunTypeTimed // most conservative choice
	}
	if config.SoftwareName == "" {
		config.SoftwareName = s.SoftwareName()
	}
	if config.SoftwareVersion == "" {
		config.SoftwareVersion = s.SoftwareVersion()
	}
	if config.WebConnectivity.CategoryCodes == nil {
		config.WebConnectivity.CategoryCodes = []string{}
	}
	return client.CheckIn(ctx, *config)
}

// maybeLookupLocationContext is a wrapper for MaybeLookupLocationContext that calls
// the configurable testMaybeLookupLocationContext mock, if configured, and the
// real MaybeLookupLocationContext API otherwise.
func (s *Session) maybeLookupLocationContext(ctx context.Context) error {
	if s.testMaybeLookupLocationContext != nil {
		return s.testMaybeLookupLocationContext(ctx)
	}
	return s.MaybeLookupLocationContext(ctx)
}

// newProbeServicesClientForCheckIn is a wrapper for NewProbeServicesClientForCheckIn
// that calls the configurable testNewProbeServicesClientForCheckIn mock, if
// configured, and the real NewProbeServicesClient API otherwise.
func (s *Session) newProbeServicesClientForCheckIn(
	ctx context.Context) (sessionProbeServicesClientForCheckIn, error) {
	if s.testNewProbeServicesClientForCheckIn != nil {
		return s.testNewProbeServicesClientForCheckIn(ctx)
	}
	client, err := s.NewProbeServicesClient(ctx)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// Close ensures that we close all the idle connections that the HTTP clients
// we are currently using may have created. It will also remove the temp dir
// that contains data from this session. Not calling this function may likely
// cause memory leaks in your application because of open idle connections,
// as well as excessive usage of disk space.
func (s *Session) Close() error {
	s.closeOnce.Do(s.doClose)
	return nil
}

// doClose implements Close. This function is called just once.
func (s *Session) doClose() {
	s.httpDefaultTransport.CloseIdleConnections()
	s.resolver.CloseIdleConnections()
	s.logger.Infof("%s", s.resolver.Stats())
	if s.tunnel != nil {
		s.tunnel.Stop()
	}
	_ = os.RemoveAll(s.tempDir)
}

// GetTestHelpersByName returns the available test helpers that
// use the specified name, or false if there's none.
func (s *Session) GetTestHelpersByName(name string) ([]model.OOAPIService, bool) {
	defer s.mu.Unlock()
	s.mu.Lock()
	services, ok := s.availableTestHelpers[name]
	return services, ok
}

// DefaultHTTPClient returns the session's default HTTP client.
func (s *Session) DefaultHTTPClient() model.HTTPClient {
	return &http.Client{Transport: s.httpDefaultTransport}
}

// FetchTorTargets fetches tor targets from the API.
func (s *Session) FetchTorTargets(
	ctx context.Context, cc string) (map[string]model.OOAPITorTarget, error) {
	clnt, err := s.NewOrchestraClient(ctx)
	if err != nil {
		return nil, err
	}
	return clnt.FetchTorTargets(ctx, cc)
}

// FetchURLList fetches the URL list from the API.
func (s *Session) FetchURLList(
	ctx context.Context, config model.OOAPIURLListConfig) ([]model.OOAPIURLInfo, error) {
	clnt, err := s.NewOrchestraClient(ctx)
	if err != nil {
		return nil, err
	}
	return clnt.FetchURLList(ctx, config)
}

// KeyValueStore returns the configured key-value store.
func (s *Session) KeyValueStore() model.KeyValueStore {
	return s.kvStore
}

// Logger returns the logger used by the session.
func (s *Session) Logger() model.Logger {
	return s.logger
}

// MaybeLookupLocation is a caching location lookup call.
func (s *Session) MaybeLookupLocation() error {
	return s.MaybeLookupLocationContext(context.Background())
}

// MaybeLookupBackends is a caching OONI backends lookup call.
func (s *Session) MaybeLookupBackends() error {
	return s.MaybeLookupBackendsContext(context.Background())
}

// ErrAlreadyUsingProxy indicates that we cannot create a tunnel with
// a specific name because we already configured a proxy.
var ErrAlreadyUsingProxy = errors.New(
	"session: cannot create a new tunnel of this kind: we are already using a proxy",
)

// NewExperimentBuilder returns a new experiment builder
// for the experiment with the given name, or an error if
// there's no such experiment with the given name
func (s *Session) NewExperimentBuilder(name string) (model.ExperimentBuilder, error) {
	eb, err := newExperimentBuilder(s, name)
	if err != nil {
		return nil, err
	}
	return eb, nil
}

// NewProbeServicesClient creates a new client for talking with the
// OONI probe services. This function will benchmark the available
// probe services, and select the fastest. In case all probe services
// seem to be down, we try again applying circumvention tactics.
// This function will fail IMMEDIATELY if given a cancelled context.
func (s *Session) NewProbeServicesClient(ctx context.Context) (*probeservices.Client, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err() // helps with testing
	}
	if err := s.maybeLookupBackendsContext(ctx); err != nil {
		return nil, err
	}
	if err := s.maybeLookupLocationContext(ctx); err != nil {
		return nil, err
	}
	if s.selectedProbeServiceHook != nil {
		s.selectedProbeServiceHook(s.selectedProbeService)
	}
	return probeservices.NewClient(s, *s.selectedProbeService)
}

// NewSubmitter creates a new submitter instance.
func (s *Session) NewSubmitter(ctx context.Context) (Submitter, error) {
	psc, err := s.NewProbeServicesClient(ctx)
	if err != nil {
		return nil, err
	}
	return probeservices.NewSubmitter(psc, s.Logger()), nil
}

// NewOrchestraClient creates a new orchestra client. This client is registered
// and logged in with the OONI orchestra. An error is returned on failure.
//
// This function is DEPRECATED. New code SHOULD NOT use it. It will eventually
// be made private or entirely removed from the codebase.
func (s *Session) NewOrchestraClient(ctx context.Context) (*probeservices.Client, error) {
	clnt, err := s.NewProbeServicesClient(ctx)
	if err != nil {
		return nil, err
	}
	return s.initOrchestraClient(ctx, clnt, clnt.MaybeLogin)
}

// Platform returns the current platform. The platform is one of:
//
// - android
// - ios
// - linux
// - macos
// - windows
// - unknown
//
// When running on the iOS simulator, the returned platform is
// macos rather than ios if CGO is disabled. This is a known issue,
// that however should have a very limited impact.
func (s *Session) Platform() string {
	return platform.Name()
}

// ProbeASNString returns the probe ASN as a string.
func (s *Session) ProbeASNString() string {
	return fmt.Sprintf("AS%d", s.ProbeASN())
}

// ProbeASN returns the probe ASN as an integer.
func (s *Session) ProbeASN() uint {
	defer s.mu.Unlock()
	s.mu.Lock()
	asn := model.DefaultProbeASN
	if s.location != nil {
		asn = s.location.ASN
	}
	return asn
}

// ProbeCC returns the probe CC.
func (s *Session) ProbeCC() string {
	defer s.mu.Unlock()
	s.mu.Lock()
	cc := model.DefaultProbeCC
	if s.location != nil {
		cc = s.location.CountryCode
	}
	return cc
}

// ProbeNetworkName returns the probe network name.
func (s *Session) ProbeNetworkName() string {
	defer s.mu.Unlock()
	s.mu.Lock()
	nn := model.DefaultProbeNetworkName
	if s.location != nil {
		nn = s.location.NetworkName
	}
	return nn
}

// ProbeIP returns the probe IP.
func (s *Session) ProbeIP() string {
	defer s.mu.Unlock()
	s.mu.Lock()
	ip := model.DefaultProbeIP
	if s.location != nil {
		ip = s.location.ProbeIP
	}
	return ip
}

// ProxyURL returns the Proxy URL, or nil if not set
func (s *Session) ProxyURL() *url.URL {
	return s.proxyURL
}

// ResolverASNString returns the resolver ASN as a string
func (s *Session) ResolverASNString() string {
	return fmt.Sprintf("AS%d", s.ResolverASN())
}

// ResolverASN returns the resolver ASN
func (s *Session) ResolverASN() uint {
	defer s.mu.Unlock()
	s.mu.Lock()
	asn := model.DefaultResolverASN
	if s.location != nil {
		asn = s.location.ResolverASN
	}
	return asn
}

// ResolverIP returns the resolver IP
func (s *Session) ResolverIP() string {
	defer s.mu.Unlock()
	s.mu.Lock()
	ip := model.DefaultResolverIP
	if s.location != nil {
		ip = s.location.ResolverIP
	}
	return ip
}

// ResolverNetworkName returns the resolver network name.
func (s *Session) ResolverNetworkName() string {
	defer s.mu.Unlock()
	s.mu.Lock()
	nn := model.DefaultResolverNetworkName
	if s.location != nil {
		nn = s.location.ResolverNetworkName
	}
	return nn
}

// SoftwareName returns the application name.
func (s *Session) SoftwareName() string {
	return s.softwareName
}

// SoftwareVersion returns the application version.
func (s *Session) SoftwareVersion() string {
	return s.softwareVersion
}

// TempDir returns the temporary directory.
func (s *Session) TempDir() string {
	return s.tempDir
}

// TorArgs returns the configured extra args for the tor binary. If not set
// we will not pass in any extra arg. Applies to `-OTunnel=tor` mainly.
func (s *Session) TorArgs() []string {
	return s.torArgs
}

// TorBinary returns the configured path to the tor binary. If not set
// we will attempt to use "tor". Applies to `-OTunnel=tor` mainly.
func (s *Session) TorBinary() string {
	return s.torBinary
}

// UserAgent constructs the user agent to be used in this session.
func (s *Session) UserAgent() (useragent string) {
	useragent += s.softwareName + "/" + s.softwareVersion
	useragent += " ooniprobe-engine/" + version.Version
	return
}

// getAvailableProbeServicesUnlocked returns the available probe
// services. This function WILL NOT acquire the mu mutex, therefore,
// you MUST ensure you are using it from a locked context.
func (s *Session) getAvailableProbeServicesUnlocked() []model.OOAPIService {
	if len(s.availableProbeServices) > 0 {
		return s.availableProbeServices
	}
	return probeservices.Default()
}

func (s *Session) initOrchestraClient(
	ctx context.Context, clnt *probeservices.Client,
	maybeLogin func(ctx context.Context) error,
) (*probeservices.Client, error) {
	// The original implementation has as its only use case that we
	// were registering and logging in for sending an update regarding
	// the probe whereabouts. Yet here in probe-engine, the orchestra
	// is currently only used to fetch inputs. For this purpose, we don't
	// need to communicate any specific information. The code that will
	// perform an update used to be responsible of doing that. Now, we
	// are not using orchestra for this purpose anymore.
	meta := probeservices.Metadata{
		Platform:        "miniooni",
		ProbeASN:        "AS0",
		ProbeCC:         "ZZ",
		SoftwareName:    "miniooni",
		SoftwareVersion: "0.1.0-dev",
		SupportedTests:  []string{"web_connectivity"},
	}
	if err := clnt.MaybeRegister(ctx, meta); err != nil {
		return nil, err
	}
	if err := maybeLogin(ctx); err != nil {
		return nil, err
	}
	return clnt, nil
}

// ErrAllProbeServicesFailed indicates all probe services failed.
var ErrAllProbeServicesFailed = errors.New("all available probe services failed")

// maybeLookupBackendsContext uses testMaybeLookupBackendsContext if
// not nil, otherwise it calls MaybeLookupBackendsContext.
func (s *Session) maybeLookupBackendsContext(ctx context.Context) error {
	if s.testMaybeLookupBackendsContext != nil {
		return s.testMaybeLookupBackendsContext(ctx)
	}
	return s.MaybeLookupBackendsContext(ctx)
}

// MaybeLookupBackendsContext is like MaybeLookupBackends but with context.
func (s *Session) MaybeLookupBackendsContext(ctx context.Context) error {
	defer s.mu.Unlock()
	s.mu.Lock()
	if s.selectedProbeService != nil {
		return nil
	}
	s.queryProbeServicesCount.Add(1)
	candidates := probeservices.TryAll(ctx, s, s.getAvailableProbeServicesUnlocked())
	selected := probeservices.SelectBest(candidates)
	if selected == nil {
		return ErrAllProbeServicesFailed
	}
	s.logger.Infof("session: using probe services: %+v", selected.Endpoint)
	s.selectedProbeService = &selected.Endpoint
	s.availableTestHelpers = selected.TestHelpers
	return nil
}

// LookupLocationContext performs a location lookup. If you want memoisation
// of the results, you should use MaybeLookupLocationContext.
func (s *Session) LookupLocationContext(ctx context.Context) (*geolocate.Results, error) {
	task := geolocate.NewTask(geolocate.Config{
		Logger:    s.Logger(),
		Resolver:  s.resolver,
		UserAgent: s.UserAgent(),
	})
	return task.Run(ctx)
}

// lookupLocationContext calls testLookupLocationContext if set and
// otherwise calls LookupLocationContext.
func (s *Session) lookupLocationContext(ctx context.Context) (*geolocate.Results, error) {
	if s.testLookupLocationContext != nil {
		return s.testLookupLocationContext(ctx)
	}
	return s.LookupLocationContext(ctx)
}

// MaybeLookupLocationContext is like MaybeLookupLocation but with a context
// that can be used to interrupt this long running operation. This function
// will fail IMMEDIATELY if given a cancelled context.
func (s *Session) MaybeLookupLocationContext(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err() // helps with testing
	}
	defer s.mu.Unlock()
	s.mu.Lock()
	if s.location == nil {
		location, err := s.lookupLocationContext(ctx)
		if err != nil {
			return err
		}
		s.location = location
	}
	return nil
}

var _ model.ExperimentSession = &Session{}
