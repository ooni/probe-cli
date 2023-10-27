package loader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/checkincache"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// newCheckInRequest creates a new request from the check-in API using the given [*ProbeInfo].
func newCheckInRequest(pi *ProbeInfo) *model.OOAPICheckInConfig {
	return &model.OOAPICheckInConfig{
		Charging:        pi.Charging,
		OnWiFi:          pi.OnWiFi,
		Platform:        pi.Platform,
		ProbeASN:        pi.ProbeASN,
		ProbeCC:         pi.ProbeCC,
		RunType:         pi.RunType,
		SoftwareName:    pi.SoftwareName,
		SoftwareVersion: pi.SoftwareVersion,
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: []string{},
		},
	}
}

// callCheckIn calls the check-in API.
func (c *Client) callCheckIn(ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInResult, error) {
	// create the request URL
	URL := &url.URL{
		Scheme: "https",
		Host:   c.endpoint,
		Path:   "/api/v1/check-in",
	}

	// serialize the raw request body
	rawReqBody := must.MarshalJSON(config)
	c.logger.Debugf("raw check-in request: %s", string(rawReqBody))

	// create the HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL.String(), bytes.NewReader(rawReqBody))
	if err != nil {
		return nil, err
	}

	// perform the HTTP round trip
	resp, err := c.txp.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// handle HTTP request failures
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d %s", ErrHTTPFailure, resp.StatusCode, resp.Status)
	}

	// read the raw response body
	rawRespBody, err := netxlite.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return nil, err
	}
	c.logger.Debugf("raw check-in response: %s", string(rawRespBody))

	// parse the raw response body
	var res model.OOAPICheckInResult
	if err := json.Unmarshal(rawRespBody, &res); err != nil {
		return nil, err
	}

	// store data into the check-in cache
	if err := checkincache.Store(c.store, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
