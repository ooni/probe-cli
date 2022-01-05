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
	s := fmt.Sprintf("Bearer %s", auth.Token)
	client := c.APIClientTemplate.BuildWithAuthorization(s)
	return client.FetchResource(ctx, "/api/v1/test-list/psiphon-config")
}
