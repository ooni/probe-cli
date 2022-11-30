package probeservices

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/model"
)

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
		// Note: ooapi (the unused package in v3.14.0 that implemented automatic API
		// generation) used `category_code` (singular) here, but that's wrong.
		query.Set("category_codes", strings.Join(config.Categories, ","))
	}
	var response model.OOAPIURLListResult
	err := c.APIClientTemplate.WithBodyLogging().Build().GetJSONWithQuery(ctx,
		"/api/v1/test-list/urls", query, &response)
	if err != nil {
		return nil, err
	}
	return response.Results, nil
}
