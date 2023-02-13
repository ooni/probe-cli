package dash

import (
	"context"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/mlablocatev2"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type locateDeps interface {
	HTTPClient() *http.Client
	Logger() model.Logger
	UserAgent() string
}

func locate(ctx context.Context, deps locateDeps) (*mlablocatev2.DashResult, error) {
	client := mlablocatev2.NewClient(deps.HTTPClient(), deps.Logger(), deps.UserAgent())
	result, err := client.QueryDash(ctx)
	if err != nil {
		return nil, err
	}
	runtimex.Assert(len(result) >= 1, "too few entries")
	return result[0], nil // ~same as with locate services v1
}
