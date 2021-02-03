package geolocate

import (
	"context"
	"net/http"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/engine/httpx"
)

func ipConfigIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger Logger,
	userAgent string,
) (string, error) {
	data, err := (httpx.Client{
		BaseURL:    "https://ipconfig.io",
		HTTPClient: httpClient,
		Logger:     logger,
		UserAgent:  httpheader.CLIUserAgent(),
	}).FetchResource(ctx, "/")
	if err != nil {
		return DefaultProbeIP, err
	}
	ip := strings.Trim(string(data), "\r\n\t ")
	logger.Debugf("ipconfig: body: %s", ip)
	return ip, nil
}
