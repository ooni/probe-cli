package iplookup

//
// IP lookup using cloudflare
//

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/fallback"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// cloudflareRegex is the regex used by [lookupCloudflare].
var cloudflareRegex = regexp.MustCompile("(?:ip)=(.*)")

// cloudflareWebLookup implements fallback.Service
type cloudflareWebLookup struct {
	client *Client
}

// newCloudflareWebLookup creates a new [cloudflareWebLookup] instance.
func newCloudflareWebLookup(client *Client) *cloudflareWebLookup {
	return &cloudflareWebLookup{client}
}

var _ fallback.Service[model.AddressFamily, string] = &cloudflareWebLookup{}

// Run implements fallback.Service
func (svc *cloudflareWebLookup) Run(ctx context.Context, family model.AddressFamily) (string, error) {
	// create HTTP request
	const URL = "https://www.cloudflare.com/cdn-cgi/trace"
	req := runtimex.Try1(http.NewRequestWithContext(ctx, http.MethodGet, URL, nil))
	req.Header.Set("User-Agent", model.HTTPHeaderUserAgent)

	// send request and get response body
	data, err := svc.client.httpDo(req, family)
	if err != nil {
		return "", err
	}

	// parse the response body to obtain the IP address
	ip := strings.Trim(string(cloudflareRegex.Find(data)), "ip=")
	return ip, nil
}

// URL implements fallback.Service
func (cl *cloudflareWebLookup) URL() string {
	return "iplookup+web://cloudflare/"
}
