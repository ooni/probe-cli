package dash

//
// Code to invoke m-lab's locate API.
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/mlablocate"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// locateDeps contains the dependencies for [locate].
type locateDeps interface {
	// HTTPClient returns the HTTP client we should use.
	HTTPClient() model.HTTPClient

	// Logger returns the logger we should use.
	Logger() model.Logger

	// UserAgent returns the user agent we should use.
	UserAgent() string
}

// locate issues a query to m-lab's locate services to obtain the
// m-lab server with which to perform a DASH experiment.
func locate(ctx context.Context, deps locateDeps) (mlablocate.Result, error) {
	return mlablocate.NewClient(
		deps.HTTPClient(), deps.Logger(), deps.UserAgent()).Query(ctx, "neubot")
}
