package dash

//
// Code to invoke m-lab's locate API.
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/mlablocatev2"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// locate issues a query to m-lab's locate services to obtain the
// m-lab server with which to perform a DASH experiment.
func locate(ctx context.Context, deps dependencies) (*mlablocatev2.DashResult, error) {
	client := mlablocatev2.NewClient(deps.HTTPClient(), deps.Logger(), deps.UserAgent())
	result, err := client.QueryDash(ctx)
	if err != nil {
		return nil, err
	}
	runtimex.Assert(len(result) >= 1, "too few entries")
	return result[0], nil // ~same as with locate services v1
}
