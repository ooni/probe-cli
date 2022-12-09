package probeservices

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TODO(bassosimone): move newHTTPAPIEndpoint inside probeservices.go when
// we have converted several APIs to use httpapi instead of httpx.

// TODO(bassosimone): convert all APIs to use httpapi's API

// newHTTPAPIEndpoint is a convenience function for constructing a new
// instance of *httpapi.Endpoint based on the content of Client
func (c Client) newHTTPAPIEndpoint() *httpapi.Endpoint {
	return &httpapi.Endpoint{
		BaseURL:    c.BaseURL,
		HTTPClient: c.HTTPClient,
		Host:       c.Host,
		UserAgent:  c.UserAgent,
	}
}

// CheckIn function is called by probes asking if there are tests to be run
// The config argument contains the mandatory settings.
// Returns the list of tests to run and the URLs, on success, or an explanatory error, in case of failure.
func (c Client) CheckIn(ctx context.Context, config model.OOAPICheckInConfig) (*model.OOAPICheckInNettests, error) {
	endpoint := &httpapi.Endpoint{
		BaseURL:    c.BaseURL,
		HTTPClient: c.HTTPClient,
		Host:       c.Host,
		UserAgent:  c.UserAgent,
	}
	// TODO(bassosimone): refactor this code to be able to pass c.Logger as the logger
	// TODO(bassosimone): the logger _actually_ belongs to the endpoint!
	desc, err := httpapi.NewPOSTJSONWithJSONResponseDescriptor("/api/v1/check-in", config)
	runtimex.PanicOnError(err, "httpapi.NewPOSTJSONWithJSONResponseDescriptor failed")
	desc.AcceptEncodingGzip = true
	var response model.OOAPICheckInResult
	if err := httpapi.CallWithJSONResponse(ctx, desc, endpoint, &response); err != nil {
		return nil, err
	}
	return &response.Tests, nil
}
