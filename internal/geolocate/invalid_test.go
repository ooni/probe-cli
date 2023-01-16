package geolocate

import (
	"context"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func invalidIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger model.Logger,
	userAgent string,
	resolver model.Resolver,
) (string, error) {
	return "invalid IP", nil
}
