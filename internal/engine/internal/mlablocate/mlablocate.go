// Package mlablocate contains a locate.measurementlab.net client.
package mlablocate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/iox"
)

// Client is a locate.measurementlab.net client.
type Client struct {
	HTTPClient *http.Client
	Hostname   string
	Logger     model.Logger
	Scheme     string
	UserAgent  string
}

// NewClient creates a new locate.measurementlab.net client.
func NewClient(httpClient *http.Client, logger model.Logger, userAgent string) *Client {
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
	City    string   `json:"city"`
	Country string   `json:"country"`
	IP      []string `json:"ip"`
	FQDN    string   `json:"fqdn"`
	Site    string   `json:"site"`
}

// Query performs a locate.measurementlab.net query.
func (c *Client) Query(ctx context.Context, tool string) (Result, error) {
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
