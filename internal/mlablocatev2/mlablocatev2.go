// Package mlablocatev2 implements m-lab locate services API v2. This
// API currently only allows you to get servers for ndt7. Use the
// mlablocate package for all other m-lab tools.
package mlablocatev2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/ooni/probe-cli/v3/internal/iox"
)

const (
	// ndt7URLPath is the URL path to be used for ndt
	ndt7URLPath = "v2/nearest/ndt/ndt7"
)

var (
	// ErrRequestFailed indicates that the response is not "200 Ok"
	ErrRequestFailed = errors.New("mlablocatev2: request failed")

	// ErrEmptyResponse indicates that no hosts were returned
	ErrEmptyResponse = errors.New("mlablocatev2: empty response")
)

// Logger is the logger expected by this package.
type Logger interface {
	// Debugf formats and emits a debug message.
	Debugf(format string, v ...interface{})
}

// HTTPClient is anything that looks like an http.Client.
type HTTPClient interface {
	// Do behaves like http.Client.Do.
	Do(req *http.Request) (*http.Response, error)
}

// Client is a client for v2 of the locate services. Please use the
// NewClient factory to construct a new instance of client, otherwise
// you MUST fill all the fields marked as MANDATORY.
type Client struct {
	// HTTPClient is the MANDATORY http client to use
	HTTPClient HTTPClient

	// Hostname is the MANDATORY hostname of the mlablocate API.
	Hostname string

	// Logger is the MANDATORY logger to use.
	Logger Logger

	// Scheme is the MANDATORY scheme to use (http or https).
	Scheme string

	// UserAgent is the MANDATORY user-agent to use.
	UserAgent string
}

// NewClient creates a client for v2 of the locate services.
func NewClient(httpClient HTTPClient, logger Logger, userAgent string) Client {
	return Client{
		HTTPClient: httpClient,
		Hostname:   "locate.measurementlab.net",
		Logger:     logger,
		Scheme:     "https",
		UserAgent:  userAgent,
	}
}

// entryRecord describes one of the boxes returned by v2 of
// the locate service. It gives you the FQDN of the specific
// box along with URLs for each experiment phase. You MUST
// use the URLs directly because they contain access tokens.
type entryRecord struct {
	Machine string            `json:"machine"`
	URLs    map[string]string `json:"urls"`
}

var (
	// siteRegexp is the regexp to extract the site from the
	// machine name when the domain is a v2 domain.
	//
	// Example: mlab3-mil04.mlab-oti.measurement-lab.org.
	siteRegexp = regexp.MustCompile(
		`^(mlab[1-4]d?)-([a-z]{3}[0-9tc]{2})\.([a-z0-9-]{1,16})\.(measurement-lab\.org)$`)
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
	Results []entryRecord `json:"results"`
}

// query performs a locate.measurementlab.net query
// using v2 of the locate protocol.
func (c Client) query(ctx context.Context, path string) (resultRecord, error) {
	// TODO(bassosimone): this code should probably be
	// refactored to use the httpx package.
	URL := &url.URL{
		Scheme: c.Scheme,
		Host:   c.Hostname,
		Path:   path,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
	if err != nil {
		return resultRecord{}, err
	}
	req.Header.Add("User-Agent", c.UserAgent)
	c.Logger.Debugf("mlablocatev2: GET %s", URL.String())
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return resultRecord{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return resultRecord{}, fmt.Errorf("%w: %d", ErrRequestFailed, resp.StatusCode)
	}
	data, err := iox.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return resultRecord{}, err
	}
	c.Logger.Debugf("mlablocatev2: %s", string(data))
	var result resultRecord
	if err := json.Unmarshal(data, &result); err != nil {
		return resultRecord{}, err
	}
	return result, nil
}

// NDT7Result is the result of a v2 locate services query for ndt7.
type NDT7Result struct {
	// Hostname is an informative field containing the hostname
	// to which you're connected. Because there are access tokens,
	// you cannot use this field directly.
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
func (c Client) QueryNDT7(ctx context.Context) ([]NDT7Result, error) {
	out, err := c.query(ctx, ndt7URLPath)
	if err != nil {
		return nil, err
	}
	var result []NDT7Result
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
		r.Site = entry.Site()
		r.Hostname = url.Hostname()
		result = append(result, r)
	}
	if len(result) <= 0 {
		return nil, ErrEmptyResponse
	}
	return result, nil
}
