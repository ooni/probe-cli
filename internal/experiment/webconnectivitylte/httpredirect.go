package webconnectivitylte

import (
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/optional"
)

/*
// TODO(bassosimone): document this func
func newArchivalHTTPRequestResultWithError(trace *measurexlite.Trace, network, address, alpn string,
	req *http.Request, err error) *model.ArchivalHTTPRequestResult {
	duration := trace.TimeSince(trace.ZeroTime())
	return measurexlite.NewArchivalHTTPRequestResult(
		trace.Index(),
		duration,
		network,
		address,
		alpn,
		network, // TODO(bassosimone): get rid of this duplicate field?
		req,
		nil,
		0,
		nil,
		err,
		duration,
	)
}
*/

// httpRedirectIsRedirect returns true if the status code contains a redirect.
func httpRedirectIsRedirect(status int64) bool {
	switch status {
	case 301, 302, 307, 308:
		return true
	default:
		return false
	}
}

// httpRedirectLocation returns a possibly-empty redirect URL or an error. More specifically:
//
// 1. the redirect URL is empty and the error nil if there's no redirect in resp;
//
// 2. the redirect URL is non-empty and the error is nil if there's a redirect in resp;
//
// 3. the error is non-nil if there's an error.
//
// Note that this function MAY possibly attempt to reconstruct the full redirect URL
// in cases in which we're dealing with a partial redirect such as '/en_US/index.html'.
func httpRedirectLocation(resp *http.Response) (optional.Value[*url.URL], error) {
	if !httpRedirectIsRedirect(int64(resp.StatusCode)) {
		return optional.None[*url.URL](), nil
	}

	location, err := resp.Location()
	if err != nil {
		return optional.None[*url.URL](), err
	}

	// TODO(https://github.com/ooni/probe/issues/2628): we need to handle
	// the case where the redirect URL is incomplete
	return optional.Some(location), nil
}
