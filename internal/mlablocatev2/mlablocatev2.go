// Package mlablocatev2 implements m-lab locate services API v2.
package mlablocatev2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// ndt7URLPath is the URL path to be used for ndt
const ndt7URLPath = "v2/nearest/ndt/ndt7"

// ErrRequestFailed indicates that the response is not "200 Ok"
var ErrRequestFailed = errors.New("mlablocatev2: request failed")

// ErrEmptyResponse indicates that no hosts were returned
var ErrEmptyResponse = errors.New("mlablocatev2: empty response")

// Client is a client for v2 of the locate services. Please use the
// NewClient factory to construct a new instance of client, otherwise
// you MUST fill all the fields marked as MANDATORY.
type Client struct {
	// HTTPClient is the MANDATORY http client to use
	HTTPClient model.HTTPClient

	// Hostname is the MANDATORY hostname of the mlablocate API.
	Hostname string

	// Logger is the MANDATORY logger to use.
	Logger model.DebugLogger

	// Scheme is the MANDATORY scheme to use (http or https).
	Scheme string

	// UserAgent is the MANDATORY user-agent to use.
	UserAgent string
}

// NewClient creates a client for v2 of the locate services.
func NewClient(httpClient model.HTTPClient, logger model.DebugLogger, userAgent string) *Client {
	return &Client{
		HTTPClient: httpClient,
		Hostname:   "locate.measurementlab.net",
		Logger:     logger,
		Scheme:     "https",
		UserAgent:  userAgent,
	}
}

// entryRecord describes one of the machines returned by v2 of
// the locate service. It gives you the FQDN of the specific
// machine along with URLs for each experiment phase. You SHOULD
// use the URLs directly because they contain access tokens.
type entryRecord struct {
	// Machine is the FQDN of the machine
	Machine string `json:"machine"`

	// URLs contains the URLs to use
	URLs map[string]string `json:"urls"`
}

// siteRegexp is the regexp to extract the site from the
// machine name when the domain is a v2 domain.
//
// Example: mlab3-mil04.mlab-oti.measurement-lab.org.
var siteRegexp = regexp.MustCompile(
	`^(mlab[1-4]d?)-([a-z]{3}[0-9tc]{2})\.([a-z0-9-]{1,16})\.(measurement-lab\.org)$`,
)

// Site returns the site name. If it is not possible to determine
// the site name, we return the empty string.
func (er entryRecord) Site() string {
	m := siteRegexp.FindAllStringSubmatch(er.Machine, -1)
	if len(m) != 1 || len(m[0]) != 5 {
		return ""
	}
	return m[0][2]
}

// resultRecord is a result of a query to locate.measurementlab.net.
type resultRecord struct {
	// Results contains the results.
	Results []entryRecord `json:"results"`
}

// query queries the locate service.
func (c *Client) query(ctx context.Context, path string) (*resultRecord, error) {
	URL := &url.URL{
		Scheme: c.Scheme,
		Host:   c.Hostname,
		Path:   path,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", c.UserAgent)
	c.Logger.Debugf("mlablocatev2: GET %s", URL.String())
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%w: %d", ErrRequestFailed, resp.StatusCode)
	}
	reader := io.LimitReader(resp.Body, 1<<20)
	data, err := netxlite.ReadAllContext(ctx, reader)
	if err != nil {
		return nil, err
	}
	c.Logger.Debugf("mlablocatev2: %s", string(data))
	var result resultRecord
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// NDT7Result is the result of a v2 locate services query for ndt7.
type NDT7Result struct {
	// Hostname is an informative field containing the hostname
	// to which you're connected. Because there are access tokens,
	// you CANNOT use this field directly.
	Hostname string

	// Site is an informative field containing the site
	// to which the server belongs to.
	Site string

	// WSSDownloadURL is the WebSocket URL to be used for
	// performing a download over HTTPS. Note that the URL
	// typically includes the required access token.
	WSSDownloadURL string

	// WSSUploadURL is like WSSDownloadURL but for the upload.
	WSSUploadURL string
}

// QueryNDT7 performs a v2 locate services query for ndt7.
func (c *Client) QueryNDT7(ctx context.Context) ([]*NDT7Result, error) {
	out, err := c.query(ctx, ndt7URLPath)
	if err != nil {
		return nil, err
	}
	runtimex.Assert(out != nil, "expected non-nil out")
	var result []*NDT7Result
	for _, entry := range out.Results {
		r := NDT7Result{
			WSSDownloadURL: entry.URLs["wss:///ndt/v7/download"],
			WSSUploadURL:   entry.URLs["wss:///ndt/v7/upload"],
		}
		if r.WSSDownloadURL == "" || r.WSSUploadURL == "" {
			continue
		}
		// Implementation note: we extract the hostname from the
		// download URL, under the assumption that the download and
		// the upload URLs have the same hostname.
		url, err := url.Parse(r.WSSDownloadURL)
		if err != nil {
			continue
		}
		r.Hostname = url.Hostname()
		r.Site = entry.Site()
		result = append(result, &r)
	}
	if len(result) <= 0 {
		return nil, ErrEmptyResponse
	}
	return result, nil
}

// DashResult is the result of a v2 locate services query for dash.
type DashResult struct {
	// Hostname is an informative field containing the hostname
	// to which you're connected. Because there are access tokens,
	// you CANNOT use this field directly.
	Hostname string

	// Site is an informative field containing the site
	// to which the server belongs to.
	Site string

	// NegotiateURL is the HTTPS URL to be used for
	// performing the DASH negotiate operation. Note that the
	// URL typically includes the required access token.
	NegotiateURL string

	// BaseURL is the base URL used for the download and the
	// collect phases of dash. The token is only required during
	// the negotiate phase and we can otherwise use a base URL.
	BaseURL string
}

// dashURLPath is the URL path to be used for dash
const dashURLPath = "v2/nearest/neubot/dash"

// QueryDash performs a v2 locate services query for dash.
func (c *Client) QueryDash(ctx context.Context) ([]*DashResult, error) {
	out, err := c.query(ctx, dashURLPath)
	if err != nil {
		return nil, err
	}
	runtimex.Assert(out != nil, "expected non-nil out")
	var result []*DashResult
	for _, entry := range out.Results {
		r := DashResult{
			NegotiateURL: entry.URLs["https:///negotiate/dash"],
		}
		if r.NegotiateURL == "" {
			continue
		}
		// Implementation note: we extract the hostname from the
		// download URL, under the assumption that the download and
		// the upload URLs have the same hostname.
		url, err := url.Parse(r.NegotiateURL)
		if err != nil {
			continue
		}
		r.Hostname = url.Hostname()
		r.BaseURL = dashBaseURL(url)
		r.Site = entry.Site()
		result = append(result, &r)
	}
	if len(result) <= 0 {
		return nil, ErrEmptyResponse
	}
	return result, nil
}

// dashBaseURL obtains the dash base URL from the negotiate URL.
func dashBaseURL(URL *url.URL) string {
	out := &url.URL{
		Scheme: URL.Scheme,
		Host:   URL.Host,
		Path:   "/",
	}
	return out.String()
}
