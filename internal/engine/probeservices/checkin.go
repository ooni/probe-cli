package probeservices

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// CheckIn function is called by probes asking if there are tests to be run
// The config argument contains the mandatory settings.
// Returns the list of tests to run and the URLs, on success, or an explanatory error, in case of failure.
func (c Client) CheckIn(ctx context.Context, config model.OOAPICheckInConfig) (*model.OOAPICheckInNettests, error) {
	var response model.OOAPICheckInResult
	if err := c.APIClientTemplate.Build().PostJSON(ctx, "/api/v1/check-in", config, &response); err != nil {
		return nil, err
	}
	return &response.Tests, nil
}
