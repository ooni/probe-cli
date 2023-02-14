package dash

//
// The negotiate phase of the DASH experiment.
//

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// negotiate implements one step of the negotiate phase of dash. The original server
// had a queue to avoid allowing too many clients to run in parallel. During the negotiate
// loop, clients wait for servers to give them permission to start an experiment. Modern
// servers always authorize clients to run. Since ~2023-02-14, we will use negotiate to
// authenticate using m-lab locate v2 tokens.
func negotiate(
	ctx context.Context, fqdn string, deps dependencies) (negotiateResponse, error) {
	var negotiateResp negotiateResponse

	// marshal the request body
	data, err := json.Marshal(negotiateRequest{DASHRates: defaultRates})
	runtimex.PanicOnError(err, "json.Marshal failed")
	deps.Logger().Debugf("dash: body: %s", string(data))

	// prepare the HTTP request
	var URL url.URL
	URL.Scheme = "https"
	URL.Host = fqdn
	URL.Path = negotiatePath
	req, err := deps.NewHTTPRequestWithContext(ctx, "POST", URL.String(), bytes.NewReader(data))
	if err != nil {
		return negotiateResp, err
	}
	req.Header.Set("User-Agent", deps.UserAgent())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "")

	// issue the request and read the response
	resp, err := deps.HTTPClient().Do(req)
	if err != nil {
		return negotiateResp, err
	}
	defer resp.Body.Close()

	// make sure we fail if the response indicates a failure
	if resp.StatusCode != 200 {
		return negotiateResp, errHTTPRequestFailed
	}

	// read the response body
	data, err = netxlite.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return negotiateResp, err
	}
	deps.Logger().Debugf("dash: body: %s", string(data))

	// unmarshal the response body
	err = json.Unmarshal(data, &negotiateResp)
	if err != nil {
		return negotiateResp, err
	}

	// check whether we have been authorized
	//
	// Implementation oddity: Neubot is using an integer rather than a
	// boolean for the unchoked, with obvious semantics. I wonder why
	// I choose an integer over a boolean, given that Python does have
	// support for booleans. I don't remember 🤷.
	if negotiateResp.Authorization == "" || negotiateResp.Unchoked == 0 {
		return negotiateResp, errServerBusy
	}
	return negotiateResp, nil
}
