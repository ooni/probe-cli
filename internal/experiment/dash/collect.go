package dash

//
// The collect phase of the dash experiment.
//

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// collect implements the collect phase of the dash experiment. We send to
// the neubot/dash server the results we collected and we get back a response
// from the server.
func collect(ctx context.Context, baseURL, authorization string,
	results []clientResults, deps dependencies) error {
	// marshal our results
	data, err := json.Marshal(results)
	runtimex.PanicOnError(err, "json.Marshal failed")
	deps.Logger().Debugf("dash: body: %s", string(data))

	// prepare the HTTP request
	URL, err := url.Parse(baseURL)
	if err != nil {
		return err
	}
	URL.Path = collectPath
	req, err := deps.NewHTTPRequestWithContext(ctx, "POST", URL.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", deps.UserAgent())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)

	// send the request and get a response.
	resp, err := deps.HTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// make sure the response is successful.
	if resp.StatusCode != 200 {
		return errHTTPRequestFailed
	}

	// read, parse, and ignore the response body. Historically the
	// most userful data has always been on the server side, therefore,
	// it doesn't matter much that we're discarding server results.
	data, err = netxlite.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return err
	}
	deps.Logger().Debugf("dash: body: %s", string(data))
	var serverResults []serverResults
	return json.Unmarshal(data, &serverResults)
}
