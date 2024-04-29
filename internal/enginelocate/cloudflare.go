package enginelocate

import (
	"context"
	"net"
	"regexp"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
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
	data, err := httpclientx.GetRaw(ctx, "https://www.cloudflare.com/cdn-cgi/trace", &httpclientx.Config{
		Authorization: "", // not needed
		Client:        httpClient,
		Logger:        logger,
		UserAgent:     userAgent,
	})

	// handle the error case
	if err != nil {
		return model.DefaultProbeIP, err
	}

	// find the IP addr
	r := regexp.MustCompile("(?:ip)=(.*)")
	ip := strings.Trim(string(r.Find(data)), "ip=")

	// make sure the IP addr is valid
	if net.ParseIP(ip) == nil {
		return model.DefaultProbeIP, ErrInvalidIPAddress
	}

	// done!
	return ip, nil
}
