package iplookup

//
// Code to resolve the IP address using cloudflare
//

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// cloudflareRegex is the regex used by [lookupCloudflare].
var cloudflareRegex = regexp.MustCompile("(?:ip)=(.*)")

// lookupCloudflare performs the lookup using cloudflare.
func (c *Client) lookupCloudflare(ctx context.Context, family model.AddressFamily) (string, error) {
	// create HTTP request
	const URL = "https://www.cloudflare.com/cdn-cgi/trace"
	req := runtimex.Try1(http.NewRequestWithContext(ctx, http.MethodGet, URL, nil))
	req.Header.Set("User-Agent", model.HTTPHeaderUserAgent)

	// send request and get response body
	data, err := c.httpDo(req, family)
	if err != nil {
		return "", err
	}

	// parse the response body to obtain the IP address
	ip := strings.Trim(string(cloudflareRegex.Find(data)), "ip=")
	return ip, nil
}
