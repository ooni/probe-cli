package dash

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/model"
)

type collectDeps interface {
	HTTPClient() *http.Client
	JSONMarshal(v interface{}) ([]byte, error)
	Logger() model.Logger
	NewHTTPRequest(method string, url string, body io.Reader) (*http.Request, error)
	ReadAllContext(ctx context.Context, r io.Reader) ([]byte, error)
	UserAgent() string
}

func collect(ctx context.Context, baseURL, authorization string,
	results []clientResults, deps collectDeps) error {
	data, err := deps.JSONMarshal(results)
	if err != nil {
		return err
	}
	deps.Logger().Debugf("dash: body: %s", string(data))
	URL, err := url.Parse(baseURL)
	if err != nil {
		return err
	}
	URL.Path = collectPath
	req, err := deps.NewHTTPRequest("POST", URL.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", deps.UserAgent())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)
	resp, err := deps.HTTPClient().Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errHTTPRequestFailed
	}
	defer resp.Body.Close()
	data, err = deps.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return err
	}
	deps.Logger().Debugf("dash: body: %s", string(data))
	var serverResults []serverResults
	return json.Unmarshal(data, &serverResults)
}
