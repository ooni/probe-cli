package dash

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

type negotiateDeps interface {
	HTTPClient() *http.Client
	JSONMarshal(v interface{}) ([]byte, error)
	Logger() model.Logger
	NewHTTPRequest(method string, url string, body io.Reader) (*http.Request, error)
	ReadAllContext(ctx context.Context, r io.Reader) ([]byte, error)
	Scheme() string
	UserAgent() string
}

func negotiate(
	ctx context.Context, fqdn string, deps negotiateDeps) (negotiateResponse, error) {
	var negotiateResp negotiateResponse
	data, err := deps.JSONMarshal(negotiateRequest{DASHRates: defaultRates})
	if err != nil {
		return negotiateResp, err
	}
	deps.Logger().Debugf("dash: body: %s", string(data))
	var URL url.URL
	URL.Scheme = deps.Scheme()
	URL.Host = fqdn
	URL.Path = negotiatePath
	req, err := deps.NewHTTPRequest("POST", URL.String(), bytes.NewReader(data))
	if err != nil {
		return negotiateResp, err
	}
	req.Header.Set("User-Agent", deps.UserAgent())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "")
	resp, err := deps.HTTPClient().Do(req.WithContext(ctx))
	if err != nil {
		return negotiateResp, err
	}
	if resp.StatusCode != 200 {
		return negotiateResp, errHTTPRequestFailed
	}
	defer resp.Body.Close()
	data, err = deps.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return negotiateResp, err
	}
	deps.Logger().Debugf("dash: body: %s", string(data))
	err = json.Unmarshal(data, &negotiateResp)
	if err != nil {
		return negotiateResp, err
	}
	// Implementation oddity: Neubot is using an integer rather than a
	// boolean for the unchoked, with obvious semantics. I wonder why
	// I choose an integer over a boolean, given that Python does have
	// support for booleans. I don't remember 🤷.
	if negotiateResp.Authorization == "" || negotiateResp.Unchoked == 0 {
		return negotiateResp, errServerBusy
	}
	return negotiateResp, nil
}
