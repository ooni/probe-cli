package engine

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/platform"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/sessionresolver"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/tunnel"
	"github.com/ooni/probe-cli/v3/internal/engine/kvstore"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
	"github.com/ooni/probe-cli/v3/internal/engine/resources"
	"github.com/ooni/probe-cli/v3/internal/engine/resourcesmanager"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// SessionConfig contains the Session config
type SessionConfig struct {
	AssetsDir              string
	AvailableProbeServices []model.Service
	KVStore                KVStore
	Logger                 model.Logger
	ProxyURL               *url.URL
	SoftwareName           string
	SoftwareVersion        string
	TempDir                string
	TorArgs                []string
	TorBinary              string
}

// Session is a measurement session
type Session struct {
	assetsDir                string
	availableProbeServices   []model.Service
	availableTestHelpers     map[string][]model.Service
	byteCounter              *bytecounter.Counter
	httpDefaultTransport     netx.HTTPRoundTripper
	kvStore                  model.KeyValueStore
	location                 *geolocate.Results
	logger                   model.Logger
	proxyURL                 *url.URL
	queryProbeServicesCount  *atomicx.Int64
	resolver                 *sessionresolver.Resolver
	selectedProbeServiceHook func(*model.Service)
	selectedProbeService     *model.Service
	softwareName             string
	softwareVersion          string
	tempDir                  string
	torArgs                  []string
	torBinary                string
	tunnelMu                 sync.Mutex
	tunnelName               string
	tunnel                   tunnel.Tunnel
}

// NewSession creates a new session or returns an error
func NewSession(config SessionConfig) (*Session, error) {
	if config.AssetsDir == "" {
		return nil, errors.New("AssetsDir is empty")
	}
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
		config.KVStore = kvstore.NewMemoryKeyValueStore()
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
		assetsDir:               config.AssetsDir,
		availableProbeServices:  config.AvailableProbeServices,
		byteCounter:             bytecounter.New(),
		kvStore:                 config.KVStore,
		logger:                  config.Logger,
		proxyURL:                config.ProxyURL,
		queryProbeServicesCount: atomicx.NewInt64(),
		softwareName:            config.SoftwareName,
		softwareVersion:         config.SoftwareVersion,
		tempDir:                 tempDir,
		torArgs:                 config.TorArgs,
		torBinary:               config.TorBinary,
	}
	httpConfig := netx.Config{
		ByteCounter:  sess.byteCounter,
		BogonIsError: true,
		Logger:       sess.logger,
		ProxyURL:     config.ProxyURL,
	}
	sess.resolver = &sessionresolver.Resolver{
		ByteCounter: sess.byteCounter,
		KVStore:     config.KVStore,
		Logger:      sess.logger,
		ProxyURL:    config.ProxyURL,
	}
	httpConfig.FullResolver = sess.resolver
	sess.httpDefaultTransport = netx.NewHTTPTransport(httpConfig)
	return sess, nil
}

// ASNDatabasePath returns the path where the ASN database path should
// be if you have called s.FetchResourcesIdempotent.
func (s *Session) ASNDatabasePath() string {
	return filepath.Join(s.assetsDir, resources.ASNDatabaseName)
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

// Close ensures that we close all the idle connections that the HTTP clients
// we are currently using may have created. It will also remove the temp dir
// that contains data from this session. Not calling this function may likely
// cause memory leaks in your application because of open idle connections,
// as well as excessive usage of disk space.
func (s *Session) Close() error {
	s.httpDefaultTransport.CloseIdleConnections()
	s.resolver.CloseIdleConnections()
	s.logger.Infof("%s", s.resolver.Stats())
	if s.tunnel != nil {
		s.tunnel.Stop()
	}
	return os.RemoveAll(s.tempDir)
}

// CountryDatabasePath is like ASNDatabasePath but for the country DB path.
func (s *Session) CountryDatabasePath() string {
	return filepath.Join(s.assetsDir, resources.CountryDatabaseName)
}

// GetTestHelpersByName returns the available test helpers that
// use the specified name, or false if there's none.
func (s *Session) GetTestHelpersByName(name string) ([]model.Service, bool) {
	services, ok := s.availableTestHelpers[name]
	return services, ok
}

// DefaultHTTPClient returns the session's default HTTP client.
func (s *Session) DefaultHTTPClient() *http.Client {
	return &http.Client{Transport: s.httpDefaultTransport}
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
	return s.maybeLookupBackends(context.Background())
}

// MaybeLookupBackendsContext is like MaybeLookupBackends but with context.
func (s *Session) MaybeLookupBackendsContext(ctx context.Context) (err error) {
	return s.maybeLookupBackends(ctx)
}

// ErrAlreadyUsingProxy indicates that we cannot create a tunnel with
// a specific name because we already configured a proxy.
var ErrAlreadyUsingProxy = errors.New(
	"session: cannot create a new tunnel of this kind: we are already using a proxy",
)

// MaybeStartTunnel starts the requested tunnel.
//
// This function silently succeeds if we're already using a tunnel with
// the same name or if the requested tunnel name is the empty string. This
// function fails, tho, when we already have a proxy or a tunnel with
// another name and we try to open a tunnel. This function of course also
// fails if we cannot start the requested tunnel. All in all, if you request
// for a tunnel name that is not the empty string and you get a nil error,
// you can be confident that session.ProxyURL() gives you the tunnel URL.
//
// The tunnel will be closed by session.Close().
func (s *Session) MaybeStartTunnel(ctx context.Context, name string) error {
	s.tunnelMu.Lock()
	defer s.tunnelMu.Unlock()
	if s.tunnel != nil && s.tunnelName == name {
		// We've been asked more than once to start the same tunnel.
		return nil
	}
	if s.proxyURL != nil && name == "" {
		// The user configured a proxy and here we're not actually trying
		// to start any tunnel since `name` is empty.
		return nil
	}
	if s.proxyURL != nil || s.tunnel != nil {
		// We already have a proxy or we have a different tunnel. Because a tunnel
		// sets a proxy, the second check for s.tunnel is for robustness.
		return ErrAlreadyUsingProxy
	}
	tunnel, err := tunnel.Start(ctx, tunnel.Config{
		Name:    name,
		Session: s,
	})
	if err != nil {
		s.logger.Warnf("cannot start tunnel: %+v", err)
		return err
	}
	// Implementation note: tunnel _may_ be NIL here if name is ""
	if tunnel == nil {
		return nil
	}
	s.tunnelName = name
	s.tunnel = tunnel
	s.proxyURL = tunnel.SOCKS5ProxyURL()
	return nil
}

// NewExperimentBuilder returns a new experiment builder
// for the experiment with the given name, or an error if
// there's no such experiment with the given name
func (s *Session) NewExperimentBuilder(name string) (*ExperimentBuilder, error) {
	return newExperimentBuilder(s, name)
}

// NewProbeServicesClient creates a new client for talking with the
// OONI probe services. This function will benchmark the available
// probe services, and select the fastest. In case all probe services
// seem to be down, we try again applying circumvention tactics.
func (s *Session) NewProbeServicesClient(ctx context.Context) (*probeservices.Client, error) {
	if err := s.maybeLookupBackends(ctx); err != nil {
		return nil, err
	}
	if err := s.MaybeLookupLocationContext(ctx); err != nil {
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
func (s *Session) NewOrchestraClient(ctx context.Context) (model.ExperimentOrchestraClient, error) {
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
	asn := geolocate.DefaultProbeASN
	if s.location != nil {
		asn = s.location.ASN
	}
	return asn
}

// ProbeCC returns the probe CC.
func (s *Session) ProbeCC() string {
	cc := geolocate.DefaultProbeCC
	if s.location != nil {
		cc = s.location.CountryCode
	}
	return cc
}

// ProbeNetworkName returns the probe network name.
func (s *Session) ProbeNetworkName() string {
	nn := geolocate.DefaultProbeNetworkName
	if s.location != nil {
		nn = s.location.NetworkName
	}
	return nn
}

// ProbeIP returns the probe IP.
func (s *Session) ProbeIP() string {
	ip := geolocate.DefaultProbeIP
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
	asn := geolocate.DefaultResolverASN
	if s.location != nil {
		asn = s.location.ResolverASN
	}
	return asn
}

// ResolverIP returns the resolver IP
func (s *Session) ResolverIP() string {
	ip := geolocate.DefaultResolverIP
	if s.location != nil {
		ip = s.location.ResolverIP
	}
	return ip
}

// ResolverNetworkName returns the resolver network name.
func (s *Session) ResolverNetworkName() string {
	nn := geolocate.DefaultResolverNetworkName
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

// MaybeUpdateResources updates the resources if needed.
func (s *Session) MaybeUpdateResources(ctx context.Context) error {
	return (&resourcesmanager.CopyWorker{DestDir: s.assetsDir}).Ensure()
}

func (s *Session) getAvailableProbeServices() []model.Service {
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

// LookupASN maps an IP address to its ASN and network name. This method implements
// LocationLookupASNLookupper.LookupASN.
func (s *Session) LookupASN(dbPath, ip string) (uint, string, error) {
	return geolocate.LookupASN(dbPath, ip)
}

// ErrAllProbeServicesFailed indicates all probe services failed.
var ErrAllProbeServicesFailed = errors.New("all available probe services failed")

func (s *Session) maybeLookupBackends(ctx context.Context) error {
	// TODO(bassosimone): do we need a mutex here?
	if s.selectedProbeService != nil {
		return nil
	}
	s.queryProbeServicesCount.Add(1)
	candidates := probeservices.TryAll(ctx, s, s.getAvailableProbeServices())
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
	// Implementation note: we don't perform the lookup of the resolver IP
	// when we are using a proxy because that might leak information.
	task := geolocate.Must(geolocate.NewTask(geolocate.Config{
		EnableResolverLookup: s.proxyURL == nil,
		HTTPClient:           s.DefaultHTTPClient(),
		Logger:               s.Logger(),
		ResourcesManager:     s,
		UserAgent:            s.UserAgent(),
	}))
	return task.Run(ctx)
}

// MaybeLookupLocationContext is like MaybeLookupLocation but with a context
// that can be used to interrupt this long running operation.
func (s *Session) MaybeLookupLocationContext(ctx context.Context) error {
	if s.location == nil {
		location, err := s.LookupLocationContext(ctx)
		if err != nil {
			return err
		}
		s.location = location
	}
	return nil
}

var _ model.ExperimentSession = &Session{}
