package probeservices

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// CheckIn function is called by probes asking if there are tests to be run
// The config argument contains the mandatory settings.
// Returns the list of tests to run and the URLs, on success,
// or an explanatory error, in case of failure.
func (c Client) CheckIn(
	ctx context.Context, config model.OOAPICheckInConfig) (*model.OOAPICheckInNettests, error) {
	epnt := c.newHTTPAPIEndpoint()
	desc, err := httpapi.NewPOSTJSONWithJSONResponseDescriptor("/api/v1/check-in", config)
	// Implementation note: NewPOSTJSONWithJSONResponseDescriptor fails IFF the config type
	// is not JSON serializable. Because it is, we do not expect any failure here.
	runtimex.PanicOnError(err, "httpapi.NewPOSTJSONWithJSONResponseDescriptor failed")
	// The response is potentially hundred of kilobytes, so we really want
	// it to travel compressed over the network.
	desc.AcceptEncodingGzip = true
	var response model.OOAPICheckInResult
	if err := httpapi.CallWithJSONResponse(ctx, desc, epnt, &response); err != nil {
		return nil, err
	}
	return &response.Tests, nil
}
