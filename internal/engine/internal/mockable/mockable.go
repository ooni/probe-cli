// Package mockable contains mockable objects
package mockable

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/kvstore"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/psiphonx"
	"github.com/ooni/probe-cli/v3/internal/engine/runtimex"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/torx"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/tunnel"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices/testorchestra"
)

// Session allows to mock sessions.
type Session struct {
	MockableASNDatabasePath      string
	MockableTestHelpers          map[string][]model.Service
	MockableHTTPClient           *http.Client
	MockableLogger               model.Logger
	MockableMaybeResolverIP      string
	MockableOrchestraClient      model.ExperimentOrchestraClient
	MockableOrchestraClientError error
	MockableProbeASNString       string
	MockableProbeCC              string
	MockableProbeIP              string
	MockableProbeNetworkName     string
	MockableProxyURL             *url.URL
	MockableResolverIP           string
	MockableSoftwareName         string
	MockableSoftwareVersion      string
	MockableTempDir              string
	MockableTorArgs              []string
	MockableTorBinary            string
	MockableUserAgent            string
}

// ASNDatabasePath implements ExperimentSession.ASNDatabasePath
func (sess *Session) ASNDatabasePath() string {
	return sess.MockableASNDatabasePath
}

// GetTestHelpersByName implements ExperimentSession.GetTestHelpersByName
func (sess *Session) GetTestHelpersByName(name string) ([]model.Service, bool) {
	services, okay := sess.MockableTestHelpers[name]
	return services, okay
}

// DefaultHTTPClient implements ExperimentSession.DefaultHTTPClient
func (sess *Session) DefaultHTTPClient() *http.Client {
	return sess.MockableHTTPClient
}

// KeyValueStore returns the configured key-value store.
func (sess *Session) KeyValueStore() model.KeyValueStore {
	return kvstore.NewMemoryKeyValueStore()
}

// Logger implements ExperimentSession.Logger
func (sess *Session) Logger() model.Logger {
	return sess.MockableLogger
}

// MaybeResolverIP implements ExperimentSession.MaybeResolverIP.
func (sess *Session) MaybeResolverIP() string {
	return sess.MockableMaybeResolverIP
}

// NewOrchestraClient implements ExperimentSession.NewOrchestraClient
func (sess *Session) NewOrchestraClient(ctx context.Context) (model.ExperimentOrchestraClient, error) {
	if sess.MockableOrchestraClient != nil {
		return sess.MockableOrchestraClient, nil
	}
	if sess.MockableOrchestraClientError != nil {
		return nil, sess.MockableOrchestraClientError
	}
	clnt, err := probeservices.NewClient(sess, model.Service{
		Address: "https://ams-pg-test.ooni.org/",
		Type:    "https",
	})
	runtimex.PanicOnError(err, "orchestra.NewClient should not fail here")
	meta := testorchestra.MetadataFixture()
	if err := clnt.MaybeRegister(ctx, meta); err != nil {
		return nil, err
	}
	if err := clnt.MaybeLogin(ctx); err != nil {
		return nil, err
	}
	return clnt, nil
}

// ProbeASNString implements ExperimentSession.ProbeASNString
func (sess *Session) ProbeASNString() string {
	return sess.MockableProbeASNString
}

// ProbeCC implements ExperimentSession.ProbeCC
func (sess *Session) ProbeCC() string {
	return sess.MockableProbeCC
}

// ProbeIP implements ExperimentSession.ProbeIP
func (sess *Session) ProbeIP() string {
	return sess.MockableProbeIP
}

// ProbeNetworkName implements ExperimentSession.ProbeNetworkName
func (sess *Session) ProbeNetworkName() string {
	return sess.MockableProbeNetworkName
}

// ProxyURL implements ExperimentSession.ProxyURL
func (sess *Session) ProxyURL() *url.URL {
	return sess.MockableProxyURL
}

// ResolverIP implements ExperimentSession.ResolverIP
func (sess *Session) ResolverIP() string {
	return sess.MockableResolverIP
}

// SoftwareName implements ExperimentSession.SoftwareName
func (sess *Session) SoftwareName() string {
	return sess.MockableSoftwareName
}

// SoftwareVersion implements ExperimentSession.SoftwareVersion
func (sess *Session) SoftwareVersion() string {
	return sess.MockableSoftwareVersion
}

// TempDir implements ExperimentSession.TempDir
func (sess *Session) TempDir() string {
	return sess.MockableTempDir
}

// TorArgs implements ExperimentSession.TorArgs.
func (sess *Session) TorArgs() []string {
	return sess.MockableTorArgs
}

// TorBinary implements ExperimentSession.TorBinary.
func (sess *Session) TorBinary() string {
	return sess.MockableTorBinary
}

// UserAgent implements ExperimentSession.UserAgent
func (sess *Session) UserAgent() string {
	return sess.MockableUserAgent
}

var _ model.ExperimentSession = &Session{}
var _ probeservices.Session = &Session{}
var _ psiphonx.Session = &Session{}
var _ tunnel.Session = &Session{}
var _ torx.Session = &Session{}

// ExperimentOrchestraClient is the experiment's view of
// a client for querying the OONI orchestra.
type ExperimentOrchestraClient struct {
	MockableFetchPsiphonConfigResult []byte
	MockableFetchPsiphonConfigErr    error
	MockableFetchTorTargetsResult    map[string]model.TorTarget
	MockableFetchTorTargetsErr       error
	MockableFetchURLListResult       []model.URLInfo
	MockableFetchURLListErr          error
}

// FetchPsiphonConfig implements ExperimentOrchestraClient.FetchPsiphonConfig
func (c ExperimentOrchestraClient) FetchPsiphonConfig(
	ctx context.Context) ([]byte, error) {
	return c.MockableFetchPsiphonConfigResult, c.MockableFetchPsiphonConfigErr
}

// FetchTorTargets implements ExperimentOrchestraClient.TorTargets
func (c ExperimentOrchestraClient) FetchTorTargets(
	ctx context.Context, cc string) (map[string]model.TorTarget, error) {
	return c.MockableFetchTorTargetsResult, c.MockableFetchTorTargetsErr
}

// FetchURLList implements ExperimentOrchestraClient.FetchURLList.
func (c ExperimentOrchestraClient) FetchURLList(
	ctx context.Context, config model.URLListConfig) ([]model.URLInfo, error) {
	return c.MockableFetchURLListResult, c.MockableFetchURLListErr
}

var _ model.ExperimentOrchestraClient = ExperimentOrchestraClient{}
