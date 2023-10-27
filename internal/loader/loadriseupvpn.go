package loader

import "context"

// LoadRiseupVPN loads the RiseupVPN [*ExperimentSpec].
func (c *Client) LoadRiseupVPN(ctx context.Context, pi *ProbeInfo) (*ExperimentSpec, error) {
	// make sure that the feature flags are fresh
	if err := c.refreshFeatureFlags(ctx, pi); err != nil {
		return nil, err
	}

	// note: the registry will refuse to instantiate riseupvpn
	// unless we've been authorized by the check-in API
	spec := &ExperimentSpec{
		Name: "riseupvpn",
		Targets: []ExperimentTarget{{
			Options:      map[string]any{},
			Input:        "",
			CategoryCode: "MISC",
			CountryCode:  "ZZ",
		}},
	}
	return spec, nil
}
