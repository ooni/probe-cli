package geolocate

import (
	"context"
	"net/http"
)

func invalidIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger Logger,
	userAgent string,
) (string, error) {
	return "invalid IP", nil
}
