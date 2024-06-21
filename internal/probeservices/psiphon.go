package probeservices

import (
	"context"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/urlx"
)

// FetchPsiphonConfig fetches psiphon config from authenticated OONI orchestra.
func (c Client) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	// get credentials and authentication token
	_, auth, err := c.GetCredsAndAuth()
	if err != nil {
		return nil, err
	}

	// format Authorization header value
	s := fmt.Sprintf("Bearer %s", auth.Token)

	// construct the URL to use
	URL, err := urlx.ResolveReference(c.BaseURL, "/api/v1/test-list/psiphon-config", "")
	if err != nil {
		return nil, err
	}

	// get response
	//
	// use a model.DiscardLogger to avoid logging config
	return httpclientx.GetRaw(
		ctx,
		httpclientx.NewEndpoint(URL).WithHostOverride(c.Host),
		&httpclientx.Config{
			Authorization: s,
			Client:        c.HTTPClient,
			Logger:        model.DiscardLogger,
			UserAgent:     c.UserAgent,
		})
}
