// Package mlablocate contains a locate.measurementlab.net client
// implementing v1 of the locate API. This version of the API isn't
// suitable for requesting servers for ndt7. You should use the
// mlablocatev2 package for that.
package mlablocate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
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

// Client is a locate.measurementlab.net client. Please use the
// NewClient factory to construct a new instance of client, otherwise
// you MUST fill all the fields marked as MANDATORY.
type Client struct {
	// HTTPClient is the MANDATORY http client to use.
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

// NewClient creates a new locate.measurementlab.net client.
func NewClient(httpClient HTTPClient, logger Logger, userAgent string) *Client {
	return &Client{
		HTTPClient: httpClient,
		Hostname:   "locate.measurementlab.net",
		Logger:     logger,
		Scheme:     "https",
		UserAgent:  userAgent,
	}
}

// Result is a result of a query to locate.measurementlab.net.
type Result struct {
	// FQDN is the mlab server's FQDN.
	FQDN string `json:"fqdn"`

	// Site is the ID of the site where the server is.
	Site string `json:"site"`
}

// Query performs a locate.measurementlab.net query. This function returns
// either valid result, on success, or an error, on failure.
// (Note thay you cannot use this API to query for ndt7 servers. You should
// use the mlablocatev2 API to obtain such servers.)
func (c *Client) Query(ctx context.Context, tool string) (Result, error) {
	// TODO(bassosimone): this code should probably be
	// refactored to use the httpx package.
	URL := &url.URL{
		Scheme: c.Scheme,
		Host:   c.Hostname,
		Path:   tool,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Add("User-Agent", c.UserAgent)
	c.Logger.Debugf("mlablocate: GET %s", URL.String())
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Result{}, fmt.Errorf("mlablocate: non-200 status code: %d", resp.StatusCode)
	}
	data, err := iox.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return Result{}, err
	}
	c.Logger.Debugf("mlablocate: %s", string(data))
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return Result{}, err
	}
	if result.FQDN == "" {
		return Result{}, errors.New("mlablocate: returned empty FQDN")
	}
	return result, nil
}
