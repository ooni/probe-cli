package geolocate

import (
	"context"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/httpheader"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/httpx"
)

type ipInfoResponse struct {
	IP string `json:"ip"`
}

func ipInfoIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger Logger,
	userAgent string,
) (string, error) {
	var v ipInfoResponse
	err := (httpx.Client{
		Accept:     "application/json",
		BaseURL:    "https://ipinfo.io",
		HTTPClient: httpClient,
		Logger:     logger,
		UserAgent:  httpheader.CLIUserAgent(), // we must be a CLI client
	}).GetJSON(ctx, "/", &v)
	if err != nil {
		return DefaultProbeIP, err
	}
	return v.IP, nil
}
