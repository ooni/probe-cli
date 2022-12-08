package probeservices

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// GetTestHelpers is like GetCollectors but for test helpers.
func (c Client) GetTestHelpers(
	ctx context.Context) (output map[string][]model.OOAPIService, err error) {
	endpoint := &httpapi.Endpoint{
		BaseURL:    c.BaseURL,
		HTTPClient: c.HTTPClient,
		Host:       c.Host,
		UserAgent:  c.UserAgent,
	}
	// TODO(bassosimone): refactor this code to be able to pass c.Logger as the logger
	desc := httpapi.NewGETJSONDescriptor(log.Log, "/api/v1/test-helpers")
	runtimex.PanicOnError(err, "httpapi.NewPOSTJSONWithJSONResponseDescriptor failed")
	desc.AcceptEncodingGzip = true
	err = httpapi.CallWithJSONResponse(ctx, desc, endpoint, &output)
	return
}
