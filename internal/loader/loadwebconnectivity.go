package loader

import (
	"context"
)

// WebConnectivityQuery contains the query to load the Web Connectivity [*ExperimentSpec].
type WebConnectivityQuery struct {
	ProbeInfo     ProbeInfo `json:"probe_info"`
	CategoryCodes []string  `json:"category_codes"`
}

// LoadWebConnectivity loads the Web Connectivity [*ExperimentSpec].
func (c *Client) LoadWebConnectivity(ctx context.Context, query *WebConnectivityQuery) (*ExperimentSpec, error) {
	// create the request for the check-in API.
	req := newCheckInRequest(&query.ProbeInfo)
	req.WebConnectivity.CategoryCodes = query.CategoryCodes

	// call the check-in API
	resp, err := c.callCheckIn(ctx, req)
	if err != nil {
		return nil, err
	}

	// handle the case where there are no targets
	if resp.Tests.WebConnectivity == nil {
		return nil, ErrNoTargets
	}

	// create the experiment spec
	spec := &ExperimentSpec{
		Name:    "web_connectivity",
		Targets: []ExperimentTarget{},
	}
	for _, entry := range resp.Tests.WebConnectivity.URLs {
		spec.Targets = append(spec.Targets, ExperimentTarget{
			Options:      map[string]any{},
			Input:        entry.URL,
			CategoryCode: entry.CategoryCode,
			CountryCode:  entry.CountryCode,
		})
	}

	return spec, nil
}
