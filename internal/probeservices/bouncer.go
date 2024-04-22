package probeservices

//
// bouncer.go - GET /api/v1/test-helpers
//

import (
	"context"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// GetTestHelpers queries the /api/v1/test-helpers API.
func (c *Client) GetTestHelpers(ctx context.Context) (map[string][]model.OOAPIService, error) {
	// construct the URL to use
	URL, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	URL.Path = "/api/v1/test-helpers"

	// get the response
	return httpclientx.GetJSON[map[string][]model.OOAPIService](
		ctx, URL.String(), c.HTTPClient, c.Logger, c.UserAgent)
}
