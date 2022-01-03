package probeservices

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/model"
)

type urlListResult struct {
	Results []model.OOAPIURLInfo `json:"results"`
}

// FetchURLList fetches the list of URLs used by WebConnectivity. The config
// argument contains the optional settings. Returns the list of URLs, on success,
// or an explanatory error, in case of failure.
func (c Client) FetchURLList(ctx context.Context, config model.OOAPIURLListConfig) ([]model.OOAPIURLInfo, error) {
	query := url.Values{}
	if config.CountryCode != "" {
		query.Set("country_code", config.CountryCode)
	}
	if config.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", config.Limit))
	}
	if len(config.Categories) > 0 {
		query.Set("category_codes", strings.Join(config.Categories, ","))
	}
	var response urlListResult
	err := c.Client.GetJSONWithQuery(ctx, "/api/v1/test-list/urls", query, &response)
	if err != nil {
		return nil, err
	}
	return response.Results, nil
}
