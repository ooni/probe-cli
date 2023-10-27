package loader

import (
	"context"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/checkincache"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// Client is a client to load OONI experiments.
//
// The zero value is invalid. Please, use [NewClient] to construct.
type Client struct {
	// endpoint is the probe services endpoint to use.
	endpoint string

	// logger is the logger to use.
	logger model.Logger

	// store is the key-value store to use.
	store model.KeyValueStore

	// txp is the transport to use.
	txp model.HTTPTransport
}

// NewClient creates a new [*Client].
//
// Arguments:
//
// - endpoint is the endpoint used by the probe services;
//
// - logger is the logger to use;
//
// - store is the key-value store to use;
//
// - txp is the HTTP transport to use.
//
// This constructor BORROWS the HTTP transport. It would be the caller's
// responsibility to CloseIdleConnections if/when needed.
func NewClient(endpoint string, logger model.Logger, store model.KeyValueStore, txp model.HTTPTransport) *Client {
	return &Client{
		endpoint: endpoint,
		logger:   logger,
		store:    store,
		txp:      txp,
	}
}

// ErrNoTargets indicates that there are no targets for the given experiment.
var ErrNoTargets = errors.New("loader: no targets for experiment")

// ErrHTTPFailure indicates that there was an HTTP failure.
var ErrHTTPFailure = errors.New("loader: HTTP request failed")

// errUnauthorized indicates that the server returned 401.
var errUnauthorized = errors.New("unauthorized")

// refreshFeatureFlags ensures that the feature flags are fresh.
func (c *Client) refreshFeatureFlags(ctx context.Context, pi *ProbeInfo) error {
	// did the feature flags expire?
	flags, err := checkincache.GetFeatureFlagsWrapper(c.store)
	if err == nil && !flags.DidExpire() {
		return nil
	}
	// create the request for the check-in API.
	req := newCheckInRequest(pi)

	// call the check-in API
	if _, err := c.callCheckIn(ctx, req); err != nil {
		return err
	}

	// assume we refreshed the flags as a side effect of calling
	// the check-in API above and continue
	return nil
}
