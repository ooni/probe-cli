package dash

//
// The collect phase of the dash experiment.
//

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// collectDeps contains the dependencies for the [collect] function.
type collectDeps interface {
	// HTTPClient returns the HTTP client to use.
	HTTPClient() model.HTTPClient

	// JSONMarshal allows to mock the [json.Marshal] function.
	JSONMarshal(v any) ([]byte, error)

	// Logger returns the logger we should use.
	Logger() model.Logger

	// NewHTTPRequestWithContext allows to mock the [http.NewRequestWithContext] function.
	NewHTTPRequestWithContext(
		context context.Context, method string, url string, body io.Reader) (*http.Request, error)

	// RealAllContext allows to mock the [netxlite.ReadAllContext] function.
	ReadAllContext(ctx context.Context, r io.Reader) ([]byte, error)

	// Scheme returns the scheme we should use.
	Scheme() string

	// UserAgent returns the user agent we should use.
	UserAgent() string
}

// collect implements the collect phase of the dash experiment. We send to
// the neubot/dash server the results we collected and we get back a response
// from the server.
func collect(ctx context.Context, fqdn, authorization string,
	results []clientResults, deps collectDeps) error {
	// marshal our results
	data, err := deps.JSONMarshal(results)
	if err != nil {
		return err
	}
	deps.Logger().Debugf("dash: body: %s", string(data))

	// prepare the HTTP request
	var URL url.URL
	URL.Scheme = deps.Scheme()
	URL.Host = fqdn
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
	data, err = deps.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return err
	}
	deps.Logger().Debugf("dash: body: %s", string(data))
	var serverResults []serverResults
	return json.Unmarshal(data, &serverResults)
}
