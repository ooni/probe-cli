package probeservices

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/urlx"
)

// FetchTorTargets returns the targets for the tor experiment.
func (c Client) FetchTorTargets(ctx context.Context, cc string) (map[string]model.OOAPITorTarget, error) {
	// get credentials and authentication token
	_, auth, err := c.GetCredsAndAuth()
	if err != nil {
		return nil, err
	}

	// format Authorization header value
	s := fmt.Sprintf("Bearer %s", auth.Token)

	// create query string
	query := url.Values{}
	query.Add("country_code", cc)

	// construct the URL to use
	URL, err := urlx.ResolveReference(c.BaseURL, "/api/v1/test-list/tor-targets", query.Encode())
	if err != nil {
		return nil, err
	}

	// get response
	//
	// use a model.DiscardLogger to avoid logging bridges
	return httpclientx.GetJSON[map[string]model.OOAPITorTarget](
		ctx,
		httpclientx.NewBaseURL(URL).WithHostOverride(c.Host),
		&httpclientx.Config{
			Authorization: s,
			Client:        c.HTTPClient,
			Logger:        model.DiscardLogger,
			UserAgent:     c.UserAgent,
		})
}
