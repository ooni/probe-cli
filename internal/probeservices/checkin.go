package probeservices

//
// checkin.go - POST /api/v1/check-in
//

import (
	"context"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// CheckIn function is called by probes asking if there are tests to be run
// The config argument contains the mandatory settings.
// This function will additionally update the [checkincache] such that we
// track selected parts of the check-in API response.
// Returns the list of tests to run and the URLs, on success,
// or an explanatory error, in case of failure.
func (c *Client) CheckIn(ctx context.Context, input model.OOAPICheckInConfig) (*model.OOAPICheckInResult, error) {
	// construct the URL to use
	URL, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	URL.Path = "/api/v1/check-in"

	// get the response
	return httpclientx.PostJSON[*model.OOAPICheckInConfig, *model.OOAPICheckInResult](
		ctx, URL.String(), c.HTTPClient, &input, c.Logger, c.UserAgent)
}
