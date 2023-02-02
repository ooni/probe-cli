package dash

import (
	"context"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/mlablocate"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type locateDeps interface {
	HTTPClient() *http.Client
	Logger() model.Logger
	UserAgent() string
}

func locate(ctx context.Context, deps locateDeps) (mlablocate.Result, error) {
	return mlablocate.NewClient(
		deps.HTTPClient(), deps.Logger(), deps.UserAgent()).Query(ctx, "neubot")
}
