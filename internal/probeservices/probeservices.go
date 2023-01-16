// Package probeservices contains code to contact OONI probe services.
//
// The probe services are HTTPS endpoints distributed across a bunch of data
// centres implementing a bunch of OONI APIs. When started, OONI will benchmark
// the available probe services and select the fastest one. Eventually all the
// possible OONI APIs will run as probe services.
//
// This package implements the following APIs:
//
// 1. v2.0.0 of the OONI bouncer specification defined
// in https://github.com/ooni/spec/blob/master/backends/bk-004-bouncer;
//
// 2. v2.0.0 of the OONI collector specification defined
// in https://github.com/ooni/spec/blob/master/backends/bk-003-collector.md;
//
// 3. most of the OONI orchestra API: login, register, fetch URLs for
// the Web Connectivity experiment, input for Tor and Psiphon.
//
// Orchestra is a set of OONI APIs for probe orchestration. We currently mainly
// using it for fetching inputs for the tor, psiphon, and web experiments.
//
// In addition, this package also contains code to benchmark the available
// probe services, discard non working ones, select the fastest.
package probeservices

import (
	"errors"
	"net/url"
	"sync/atomic"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

var (
	// ErrUnsupportedEndpoint indicates that we don't support this endpoint type.
	ErrUnsupportedEndpoint = errors.New("probe services: unsupported endpoint type")

	// ErrUnsupportedCloudFrontAddress indicates that we don't support this
	// cloudfront address (e.g. wrong scheme, presence of port).
	ErrUnsupportedCloudFrontAddress = errors.New(
		"probe services: unsupported cloud front address",
	)

	// ErrNotRegistered indicates that the probe is not registered
	// with the OONI orchestra backend.
	ErrNotRegistered = errors.New("not registered")

	// ErrNotLoggedIn indicates that we are not logged in
	ErrNotLoggedIn = errors.New("not logged in")

	// ErrInvalidMetadata indicates that the metadata is not valid
	ErrInvalidMetadata = errors.New("invalid metadata")
)

// Session is how this package sees a Session.
type Session interface {
	DefaultHTTPClient() model.HTTPClient
	KeyValueStore() model.KeyValueStore
	Logger() model.Logger
	ProxyURL() *url.URL
	UserAgent() string
}

// Client is a client for the OONI probe services API.
type Client struct {
	httpx.APIClientTemplate
	LoginCalls    *atomic.Int64
	RegisterCalls *atomic.Int64
	StateFile     StateFile
}

// GetCredsAndAuth is an utility function that returns the credentials with
// which we are registered and the token with which we're logged in. If we're
// not registered or not logged in, an error is returned instead.
func (c Client) GetCredsAndAuth() (*model.OOAPILoginCredentials, *model.OOAPILoginAuth, error) {
	state := c.StateFile.Get()
	creds := state.Credentials()
	if creds == nil {
		return nil, nil, ErrNotRegistered
	}
	auth := state.Auth()
	if auth == nil {
		return nil, nil, ErrNotLoggedIn
	}
	return creds, auth, nil
}

// NewClient creates a new client for the specified probe services endpoint. This
// function fails, e.g., we don't support the specified endpoint.
func NewClient(sess Session, endpoint model.OOAPIService) (*Client, error) {
	client := &Client{
		APIClientTemplate: httpx.APIClientTemplate{
			BaseURL:    endpoint.Address,
			HTTPClient: sess.DefaultHTTPClient(),
			Logger:     sess.Logger(),
			UserAgent:  sess.UserAgent(),
		},
		LoginCalls:    &atomic.Int64{},
		RegisterCalls: &atomic.Int64{},
		StateFile:     NewStateFile(sess.KeyValueStore()),
	}
	switch endpoint.Type {
	case "https":
		return client, nil
	case "cloudfront":
		// Do the cloudfronting dance. The front must appear inside of the
		// URL, so that we use it for DNS resolution and SNI. The real domain
		// must instead appear inside of the Host header.
		URL, err := url.Parse(client.BaseURL)
		if err != nil {
			return nil, err
		}
		if URL.Scheme != "https" || URL.Host != URL.Hostname() {
			return nil, ErrUnsupportedCloudFrontAddress
		}
		client.APIClientTemplate.Host = URL.Hostname()
		URL.Host = endpoint.Front
		client.BaseURL = URL.String()
		if _, err := url.Parse(client.BaseURL); err != nil {
			return nil, err
		}
		return client, nil
	default:
		return nil, ErrUnsupportedEndpoint
	}
}

// newHTTPAPIEndpoint is a convenience function for constructing a new
// instance of *httpapi.Endpoint based on the content of Client
func (c Client) newHTTPAPIEndpoint() *httpapi.Endpoint {
	// TODO(https://github.com/ooni/probe/issues/2362): we should migrate all APIs to use
	// httpapi, which supports fallback, while httpx does not support fallback.
	return &httpapi.Endpoint{
		BaseURL:    c.BaseURL,
		HTTPClient: c.HTTPClient,
		Logger:     c.Logger,
		Host:       c.Host,
		UserAgent:  c.UserAgent,
	}
}
