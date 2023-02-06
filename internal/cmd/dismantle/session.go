package main

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type experimentSession struct {
	httpClient  model.HTTPClient
	location    *geolocate.Results
	logger      model.Logger
	testHelpers map[string][]model.OOAPIService
	userAgent   string
}

var _ model.ExperimentSession = &experimentSession{}

// DefaultHTTPClient implements model.ExperimentSession
func (es *experimentSession) DefaultHTTPClient() model.HTTPClient {
	return es.httpClient
}

// FetchPsiphonConfig implements model.ExperimentSession
func (es *experimentSession) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	// FIXME: we need to call the backend API for this I think?
	panic("unimplemented")
}

// FetchTorTargets implements model.ExperimentSession
func (es *experimentSession) FetchTorTargets(ctx context.Context, cc string) (map[string]model.OOAPITorTarget, error) {
	// FIXME: we need to call the backend API for this I think?
	panic("unimplemented")
}

// GetTestHelpersByName implements model.ExperimentSession
func (es *experimentSession) GetTestHelpersByName(name string) ([]model.OOAPIService, bool) {
	value, found := es.testHelpers[name]
	return value, found
}

// Logger implements model.ExperimentSession
func (es *experimentSession) Logger() model.Logger {
	return es.logger
}

// ProbeCC implements model.ExperimentSession
func (es *experimentSession) ProbeCC() string {
	return es.location.CountryCode
}

// ResolverIP implements model.ExperimentSession
func (es *experimentSession) ResolverIP() string {
	return es.location.ResolverIP
}

// TempDir implements model.ExperimentSession
func (es *experimentSession) TempDir() string {
	panic("unimplemented") // FIXME
}

// TorArgs implements model.ExperimentSession
func (es *experimentSession) TorArgs() []string {
	panic("unimplemented") // FIXME
}

// TorBinary implements model.ExperimentSession
func (es *experimentSession) TorBinary() string {
	panic("unimplemented") // FIXME
}

// TunnelDir implements model.ExperimentSession
func (es *experimentSession) TunnelDir() string {
	panic("unimplemented") // FIXME
}

// UserAgent implements model.ExperimentSession
func (es *experimentSession) UserAgent() string {
	return es.userAgent
}
