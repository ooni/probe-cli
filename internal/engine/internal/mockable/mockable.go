// Package mockable contains mockable objects
package mockable

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
)

// Session allows to mock sessions.
type Session struct {
	MockableTestHelpers              map[string][]model.Service
	MockableHTTPClient               *http.Client
	MockableLogger                   model.Logger
	MockableMaybeResolverIP          string
	MockableProbeASNString           string
	MockableProbeCC                  string
	MockableProbeIP                  string
	MockableProbeNetworkName         string
	MockableProxyURL                 *url.URL
	MockableFetchPsiphonConfigResult []byte
	MockableFetchPsiphonConfigErr    error
	MockableFetchTorTargetsResult    map[string]model.TorTarget
	MockableFetchTorTargetsErr       error
	MockableFetchURLListResult       []model.URLInfo
	MockableFetchURLListErr          error
	MockableResolverIP               string
	MockableSoftwareName             string
	MockableSoftwareVersion          string
	MockableTempDir                  string
	MockableTorArgs                  []string
	MockableTorBinary                string
	MockableUserAgent                string
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

// FetchPsiphonConfig implements ExperimentSession.FetchPsiphonConfig
func (sess *Session) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	return sess.MockableFetchPsiphonConfigResult, sess.MockableFetchPsiphonConfigErr
}

// FetchTorTargets implements ExperimentSession.TorTargets
func (sess *Session) FetchTorTargets(
	ctx context.Context, cc string) (map[string]model.TorTarget, error) {
	return sess.MockableFetchTorTargetsResult, sess.MockableFetchTorTargetsErr
}

// FetchURLList implements ExperimentSession.FetchURLList.
func (sess *Session) FetchURLList(
	ctx context.Context, config model.URLListConfig) ([]model.URLInfo, error) {
	return sess.MockableFetchURLListResult, sess.MockableFetchURLListErr
}

// KeyValueStore returns the configured key-value store.
func (sess *Session) KeyValueStore() model.KeyValueStore {
	return &kvstore.Memory{}
}

// Logger implements ExperimentSession.Logger
func (sess *Session) Logger() model.Logger {
	return sess.MockableLogger
}

// MaybeResolverIP implements ExperimentSession.MaybeResolverIP.
func (sess *Session) MaybeResolverIP() string {
	return sess.MockableMaybeResolverIP
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
