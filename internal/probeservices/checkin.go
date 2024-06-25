package probeservices

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/checkincache"
	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/urlx"
)

// CheckIn function is called by probes asking if there are tests to be run
// The config argument contains the mandatory settings.
// This function will additionally update the [checkincache] such that we
// track selected parts of the check-in API response.
// Returns the list of tests to run and the URLs, on success,
// or an explanatory error, in case of failure.
func (c Client) CheckIn(
	ctx context.Context, config model.OOAPICheckInConfig) (*model.OOAPICheckInResult, error) {
	// construct the URL to use
	URL, err := urlx.ResolveReference(c.BaseURL, "/api/v1/check-in", "")
	if err != nil {
		return nil, err
	}

	// issue the API call
	resp, err := httpclientx.PostJSON[*model.OOAPICheckInConfig, *model.OOAPICheckInResult](
		ctx,
		httpclientx.NewBaseURL(URL).WithHostOverride(c.Host),
		&config,
		&httpclientx.Config{
			Authorization: "", // not needed
			Client:        c.HTTPClient,
			Logger:        c.Logger,
			UserAgent:     c.UserAgent,
		})

	// handle the case of error
	if err != nil {
		return nil, err
	}

	// make sure we track selected parts of the response and ignore
	// the error because OONI Probe would also work without this caching
	// it would only work more poorly, but it does not seem worth it
	// crippling it entirely if we cannot write into the kvstore
	_ = checkincache.Store(c.KVStore, resp)
	return resp, nil
}
