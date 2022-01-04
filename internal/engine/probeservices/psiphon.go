package probeservices

import (
	"context"
	"fmt"
)

// FetchPsiphonConfig fetches psiphon config from authenticated OONI orchestra.
func (c Client) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	_, auth, err := c.GetCredsAndAuth()
	if err != nil {
		return nil, err
	}
	// Note: the following code is very bad: it copies the original
	// API client and then overrides one of its fields. Bleah...
	client := c.APIClient
	client.Authorization = fmt.Sprintf("Bearer %s", auth.Token)
	return client.FetchResource(ctx, "/api/v1/test-list/psiphon-config")
}
