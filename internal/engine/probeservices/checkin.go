package probeservices

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// CheckIn function is called by probes asking if there are tests to be run
// The config argument contains the mandatory settings.
// Returns the list of tests to run and the URLs, on success, or an explanatory error, in case of failure.
func (c Client) CheckIn(ctx context.Context, config model.OOAPICheckInConfig) (*model.OOAPICheckInNettests, error) {
	// TODO(bassosimone): convert all APIs to use httpapi's API
	endpoint := &httpapi.Endpoint{
		BaseURL:    c.BaseURL,
		HTTPClient: c.HTTPClient,
		Host:       c.Host,
		UserAgent:  c.UserAgent,
	}
	// TODO(bassosimone): refactor this code to be able to pass c.Logger as the logger
	desc, err := httpapi.NewPOSTJSONWithJSONResponseDescriptor(log.Log, "/api/v1/check-in", config)
	runtimex.PanicOnError(err, "httpapi.NewPOSTJSONWithJSONResponseDescriptor failed")
	desc.AcceptEncodingGzip = true
	var response model.OOAPICheckInResult
	if err := httpapi.CallWithJSONResponse(ctx, desc, endpoint, &response); err != nil {
		return nil, err
	}
	return &response.Tests, nil
}
