package enginelocate

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func cloudflareIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger model.Logger,
	userAgent string,
	resolver model.Resolver,
) (string, error) {
	// TODO(https://github.com/ooni/probe/issues/2551)
	const timeout = 45 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	data, err := (&httpx.APIClientTemplate{
		BaseURL:    "https://www.cloudflare.com",
		HTTPClient: httpClient,
		Logger:     logger,
		UserAgent:  model.HTTPHeaderUserAgent,
	}).WithBodyLogging().Build().FetchResource(ctx, "/cdn-cgi/trace")
	if err != nil {
		return model.DefaultProbeIP, err
	}
	r := regexp.MustCompile("(?:ip)=(.*)")
	ip := strings.Trim(string(r.Find(data)), "ip=")
	logger.Debugf("cloudflare: body: %s", ip)
	return ip, nil
}
