// Package backendclient implements a client to communicate
// with the OONI backend infrastructure.
package backendclient

import (
	"context"
	"errors"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/ooapi"
)

// Config contains configuration for [New].
type Config struct {
	// BaseURL is the OPTIONAL OONI backend URL.
	BaseURL *url.URL

	// KVStore is the MANDATORY key-value store to use.
	KVStore model.KeyValueStore

	// HTTPClient is the MANDATORY underlying HTTPClient to use.
	HTTPClient model.HTTPClient

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// UserAgent is the MANDATORY user agent to use.
	UserAgent string
}

// Client is a client to communicate with the OONI backend.
type Client struct {
	// endpoint is the HTTP API endpoint.
	endpoint *httpapi.Endpoint
}

// New constructs a new instance of [Client].
func New(config *Config) *Client {
	baseURL := "https://api.ooni.io/"
	if config.BaseURL != nil {
		baseURL = config.BaseURL.String()
	}
	endpoint := &httpapi.Endpoint{
		BaseURL:    baseURL,
		HTTPClient: config.HTTPClient,
		Host:       "", // no need to configure
		Logger:     config.Logger,
		UserAgent:  config.UserAgent,
	}
	backendClient := &Client{
		endpoint: endpoint,
	}
	return backendClient
}

// CheckIn invokes the check-in API.
func (c *Client) CheckIn(
	ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInResult, error) {
	op := measurexlite.NewOperationLogger(
		c.endpoint.Logger,
		"backendclient: check-in using %s",
		c.endpoint.BaseURL,
	)
	r, err := httpapi.Call(ctx, ooapi.NewDescriptorCheckIn(config), c.endpoint)
	op.Stop(err)
	return r, err
}

// FetchPsiphonConfig retrieves Psiphon configuration.
func (c *Client) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// FetchTorTargets fetches measurement targets for the tor experiment.
func (c *Client) FetchTorTargets(
	ctx context.Context, cc string) (result map[string]model.OOAPITorTarget, err error) {
	return nil, errors.New("not implemented")
}

// Submit submits the given measurement.
func (c *Client) Submit(ctx context.Context, m *model.Measurement) error {
	req := &model.OOAPICollectorUpdateRequest{
		Format:  "json",
		Content: m,
	}
	descriptor := ooapi.NewSubmitMeasurementDescriptor(req, m.ReportID)
	_, err := httpapi.Call(ctx, descriptor, c.endpoint)
	return err
}
