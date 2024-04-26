package enginelocate

import (
	"context"
	"net"
	"regexp"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func cloudflareIPLookup(
	ctx context.Context,
	httpClient model.HTTPClient,
	logger model.Logger,
	userAgent string,
	resolver model.Resolver,
) (string, error) {
	// get the raw response body
	data, err := (&httpx.APIClientTemplate{
		BaseURL:    "https://www.cloudflare.com",
		HTTPClient: httpClient,
		Logger:     logger,
		UserAgent:  model.HTTPHeaderUserAgent,
	}).WithBodyLogging().Build().FetchResource(ctx, "/cdn-cgi/trace")

	// handle the error case
	if err != nil {
		return model.DefaultProbeIP, err
	}

	// find the IP addr
	r := regexp.MustCompile("(?:ip)=(.*)")
	ip := strings.Trim(string(r.Find(data)), "ip=")
	logger.Debugf("cloudflare: body: %s", ip)

	// make sure the IP addr is valid
	if net.ParseIP(ip) == nil {
		return model.DefaultProbeIP, ErrInvalidIPAddress
	}

	// done!
	return ip, nil
}
