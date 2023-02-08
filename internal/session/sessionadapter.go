package session

import (
	"context"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// sessionAdapter adapts [Session] to be a [model.ExperimentSesion].
type sessionAdapter struct {
	httpClient  model.HTTPClient
	location    *geolocate.Results
	logger      model.Logger
	tempDir     string
	testHelpers map[string][]model.OOAPIService
	torBinary   string
	tunnelDir   string
	userAgent   string
}

// ErrNoLocation means we cannot proceed without knowing the probe location.
var ErrNoLocation = errors.New("session: no location information")

// ErrNoCheckIn means we cannot proceed without the check-in API results.
var ErrNoCheckIn = errors.New("session: no check-in information")

// newSessionAdapter creates a new [sessionAdapter] instance.
func newSessionAdapter(state *state) (*sessionAdapter, error) {
	if state.location == nil {
		return nil, ErrNoLocation
	}
	if state.checkIn == nil {
		return nil, ErrNoCheckIn
	}
	sa := &sessionAdapter{
		httpClient:  state.httpClient,
		location:    state.location,
		logger:      state.logger,
		tempDir:     state.tempDir,
		testHelpers: state.checkIn.Conf.TestHelpers,
		torBinary:   state.torBinary,
		tunnelDir:   state.tunnelDir,
		userAgent:   state.userAgent,
	}
	return sa, nil
}

var _ model.ExperimentSession = &sessionAdapter{}

// DefaultHTTPClient implements model.ExperimentSession
func (es *sessionAdapter) DefaultHTTPClient() model.HTTPClient {
	return es.httpClient
}

// FetchPsiphonConfig implements model.ExperimentSession
func (es *sessionAdapter) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// FetchTorTargets implements model.ExperimentSession
func (es *sessionAdapter) FetchTorTargets(ctx context.Context, cc string) (map[string]model.OOAPITorTarget, error) {
	return nil, errors.New("not implemented")
}

// GetTestHelpersByName implements model.ExperimentSession
func (es *sessionAdapter) GetTestHelpersByName(name string) ([]model.OOAPIService, bool) {
	value, found := es.testHelpers[name]
	return value, found
}

// Logger implements model.ExperimentSession
func (es *sessionAdapter) Logger() model.Logger {
	return es.logger
}

// ProbeCC implements model.ExperimentSession
func (es *sessionAdapter) ProbeCC() string {
	return es.location.CountryCode
}

// ResolverIP implements model.ExperimentSession
func (es *sessionAdapter) ResolverIP() string {
	return es.location.ResolverIPAddr
}

// TempDir implements model.ExperimentSession
func (es *sessionAdapter) TempDir() string {
	return es.tempDir
}

// TorArgs implements model.ExperimentSession
func (es *sessionAdapter) TorArgs() []string {
	return []string{} // TODO(bassosimone): this field is only meaningful for bootstrap
}

// TorBinary implements model.ExperimentSession
func (es *sessionAdapter) TorBinary() string {
	return es.torBinary
}

// TunnelDir implements model.ExperimentSession
func (es *sessionAdapter) TunnelDir() string {
	return es.tunnelDir
}

// UserAgent implements model.ExperimentSession
func (es *sessionAdapter) UserAgent() string {
	return es.userAgent
}
