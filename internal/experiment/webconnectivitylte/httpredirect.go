package webconnectivitylte

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// httpRedirectIsRedirect returns whether this response is a redirect
func httpRedirectIsRedirect(resp *http.Response) bool {
	switch resp.StatusCode {
	case 301, 302, 307, 308:
		return true
	default:
		return false
	}

}

// httpValidateRedirect validates a redirect. In case of failure, the
// returned error is a [*netxlite.ErrWrapper] instance.
//
// See https://github.com/ooni/probe/issues/2628 for context.
func httpValidateRedirect(resp *http.Response) error {
	location, err := resp.Location()
	if err != nil {
		return err
	}
	if location.Host == "" {
		return &netxlite.ErrWrapper{
			Failure:    netxlite.FailureHTTPInvalidRedirectLocationHost,
			Operation:  netxlite.HTTPRoundTripOperation,
			WrappedErr: nil,
		}
	}
	return nil
}
