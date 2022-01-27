package geolocate

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func ipInfoIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger model.Logger,
	userAgent string,
) (string, error) {
	data, err := (&httpx.APIClientTemplate{
		BaseURL:    "https://www.nginx.com/cdn-cgi/trace",
		HTTPClient: httpClient,
		Logger:     logger,
		UserAgent:  httpheader.CLIUserAgent(),
	}).WithBodyLogging().Build().FetchResource(ctx, "")
	if err != nil {
		return DefaultProbeIP, err
	}
	r := regexp.MustCompile("(?:ip)=(.*)")
	ip := strings.Trim(string(r.Find(data)), "ip=")

	logger.Debugf("ipconfig: body: %s", ip)
	return ip, nil
}
