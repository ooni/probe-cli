// Package aggregationapi allows one to query the OONI aggregation API.
package aggregationapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Response is the response from the aggregation API.
type Response struct {
	Result ResponseResult `json:"result"`
	V      int64          `json:"v"`
}

// ResponseResult is the result field inside a [Response].
type ResponseResult struct {
	AnomalyCount     int64 `json:"anomaly_count"`
	ConfirmedCount   int64 `json:"confirmed_count"`
	FailureCount     int64 `json:"failure_count"`
	MeasurementCount int64 `json:"measurement_count"`
	OKCount          int64 `json:"ok_count"`
}

// Query invokes the aggregation API for the given input URL and
// returns as the result the API response. This function calls
// [runtimex.PanicOnError] in case of failure.
func Query(
	ctx context.Context,
	apiURL string,
	inputURL string,
) *Response {
	// create the query string
	oneMonthAgo := time.Now().Add(-30 * 24 * time.Hour)
	query := url.Values{}
	query.Add("since", oneMonthAgo.Format("2006-01-02T15:04:05"))
	query.Add("input", inputURL)
	query.Add("test_name", "web_connectivity")

	// create the URL to invoke the aggregation API
	URL := runtimex.Try1(url.Parse(apiURL))
	URL.Path = "/api/v1/aggregation"
	URL.RawQuery = query.Encode()

	// invoke the aggregation API using the default HTTP client.
	req := runtimex.Try1(http.NewRequestWithContext(ctx, "GET", URL.String(), nil))
	resp := runtimex.Try1(http.DefaultClient.Do(req))
	defer resp.Body.Close()
	runtimex.Assert(resp.StatusCode == 200, "aggregationapi: http request failed")

	// read the response body
	data := runtimex.Try1(netxlite.ReadAllContext(ctx, resp.Body))

	// parse the response body
	var apiResp Response
	runtimex.Try0(json.Unmarshal(data, &apiResp))

	// make sure we are using the version we expect to use
	runtimex.Assert(apiResp.V == 0, "aggregationapi: unexpected API response version")

	// return the whole JSON to the caller
	return &apiResp
}
