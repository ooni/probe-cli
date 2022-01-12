package websteps

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPDo performs the HTTP check.
// Input:
// req *http.Request
//      The same request than the one used by the Explore step.
//      This means that req contains the headers set by the original CtrlRequest, as well as,
//      in case of a redirect chain, additional headers that were added due to redirects
// transport http.RoundTripper:
//      The transport to use, either http.Transport, or http3.RoundTripper.
func HTTPDo(req *http.Request, transport http.RoundTripper) (*http.Response, []byte, error) {
	clnt := http.Client{
		CheckRedirect: func(r *http.Request, reqs []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: transport,
	}
	resp, err := clnt.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := netxlite.ReadAllContext(req.Context(), resp.Body)
	if err != nil {
		return resp, nil, nil
	}
	return resp, body, nil
}
