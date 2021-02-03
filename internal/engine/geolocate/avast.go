package geolocate

import (
	"context"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/engine/httpx"
)

type avastResponse struct {
	IP string `json:"ip"`
}

func avastIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger Logger,
	userAgent string,
) (string, error) {
	var v avastResponse
	err := (httpx.Client{
		BaseURL:    "https://ip-info.ff.avast.com",
		HTTPClient: httpClient,
		Logger:     logger,
		UserAgent:  userAgent,
	}).GetJSON(ctx, "/v1/info", &v)
	if err != nil {
		return DefaultProbeIP, err
	}
	return v.IP, nil
}
