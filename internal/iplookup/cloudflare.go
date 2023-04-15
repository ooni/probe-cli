package iplookup

//
// Code to resolve the IP address using cloudflare
//

import (
	"context"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// lookupCloudflare performs the lookup using cloudflare.
func (c *Client) lookupCloudflare(ctx context.Context, family Family) (string, error) {
	// make sure we eventually time out
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

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
	r := regexp.MustCompile("(?:ip)=(.*)")
	ip := strings.Trim(string(r.Find(data)), "ip=")

	// make sure the IP address is valid
	if net.ParseIP(ip) == nil {
		return "", ErrInvalidIPAddress
	}

	return ip, nil
}
