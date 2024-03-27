package mocks

import (
	"context"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Session allows to mock sessions.
type Session struct {
	MockGetTestHelpersByName func(name string) ([]model.OOAPIService, bool)

	MockDefaultHTTPClient func() model.HTTPClient

	MockFetchPsiphonConfig func(ctx context.Context) ([]byte, error)

	MockFetchTorTargets func(
		ctx context.Context, cc string) (map[string]model.OOAPITorTarget, error)

	MockFetchOpenVPNConfig func(
		ctx context.Context, provider, cc string) (*model.OOAPIVPNProviderConfig, error)

	MockKeyValueStore func() model.KeyValueStore

	MockLogger func() model.Logger

	MockMaybeResolverIP func() string

	MockProbeASNString func() string

	MockProbeCC func() string

	MockProbeIP func() string

	MockProbeNetworkName func() string

	MockProxyURL func() *url.URL

	MockResolverIP func() string

	MockSoftwareName func() string

	MockSoftwareVersion func() string

	MockTempDir func() string

	MockTorArgs func() []string

	MockTorBinary func() string

	MockTunnelDir func() string

	MockUserAgent func() string

	MockNewExperimentBuilder func(name string) (model.ExperimentBuilder, error)

	MockNewSubmitter func(ctx context.Context) (model.Submitter, error)

	MockCheckIn func(ctx context.Context,
		config *model.OOAPICheckInConfig) (*model.OOAPICheckInResultNettests, error)
}

func (sess *Session) GetTestHelpersByName(name string) ([]model.OOAPIService, bool) {
	return sess.MockGetTestHelpersByName(name)
}

func (sess *Session) DefaultHTTPClient() model.HTTPClient {
	return sess.MockDefaultHTTPClient()
}

func (sess *Session) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	return sess.MockFetchPsiphonConfig(ctx)
}

func (sess *Session) FetchOpenVPNConfig(
	ctx context.Context, provider, cc string) (*model.OOAPIVPNProviderConfig, error) {
	return sess.MockFetchOpenVPNConfig(ctx, provider, cc)
}

func (sess *Session) FetchTorTargets(
	ctx context.Context, cc string) (map[string]model.OOAPITorTarget, error) {
	return sess.MockFetchTorTargets(ctx, cc)
}

func (sess *Session) KeyValueStore() model.KeyValueStore {
	return sess.MockKeyValueStore()
}

func (sess *Session) Logger() model.Logger {
	return sess.MockLogger()
}

func (sess *Session) MaybeResolverIP() string {
	return sess.MockMaybeResolverIP()
}

func (sess *Session) ProbeASNString() string {
	return sess.MockProbeASNString()
}

func (sess *Session) ProbeCC() string {
	return sess.MockProbeCC()
}

func (sess *Session) ProbeIP() string {
	return sess.MockProbeIP()
}

func (sess *Session) ProbeNetworkName() string {
	return sess.MockProbeNetworkName()
}

func (sess *Session) ProxyURL() *url.URL {
	return sess.MockProxyURL()
}

func (sess *Session) ResolverIP() string {
	return sess.MockResolverIP()
}

func (sess *Session) SoftwareName() string {
	return sess.MockSoftwareName()
}

func (sess *Session) SoftwareVersion() string {
	return sess.MockSoftwareVersion()
}

func (sess *Session) TempDir() string {
	return sess.MockTempDir()
}

func (sess *Session) TorArgs() []string {
	return sess.MockTorArgs()
}

func (sess *Session) TorBinary() string {
	return sess.MockTorBinary()
}

func (sess *Session) TunnelDir() string {
	return sess.MockTunnelDir()
}

func (sess *Session) UserAgent() string {
	return sess.MockUserAgent()
}

func (sess *Session) NewExperimentBuilder(name string) (model.ExperimentBuilder, error) {
	return sess.MockNewExperimentBuilder(name)
}

func (sess *Session) NewSubmitter(ctx context.Context) (model.Submitter, error) {
	return sess.MockNewSubmitter(ctx)
}

func (sess *Session) CheckIn(ctx context.Context,
	config *model.OOAPICheckInConfig) (*model.OOAPICheckInResultNettests, error) {
	return sess.MockCheckIn(ctx, config)
}
