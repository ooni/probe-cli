package probeservices

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// FetchTorTargets returns the targets for the tor experiment.
func (c Client) FetchTorTargets(ctx context.Context, cc string) (result map[string]model.OOAPITorTarget, err error) {
	_, auth, err := c.GetCredsAndAuth()
	if err != nil {
		return nil, err
	}
	// Note: the following code is very bad: it copies the original
	// API client and then overrides one of its fields. Bleah...
	client := c.APIClient
	client.Authorization = fmt.Sprintf("Bearer %s", auth.Token)
	query := url.Values{}
	query.Add("country_code", cc)
	err = client.GetJSONWithQuery(
		ctx, "/api/v1/test-list/tor-targets", query, &result)
	return
}
