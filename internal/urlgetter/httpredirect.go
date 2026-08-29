package urlgetter

import (
	"errors"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TODO(bassosimone): this duplicates code in webconnectivitylte and we should
// instead share this code and avoid creating duplication.
//
// However, this code is slightly changed, so it's not 100% clear what to do.

// httpRedirectIsRedirect returns whether this response is a redirect
func httpRedirectIsRedirect(resp *HTTPResponse) bool {
	switch resp.Status {
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
func httpValidateRedirect(resp *HTTPResponse) error {
	if resp.Location == nil {
		return errors.New("missing location header")
	}
	if resp.Location.Host == "" {
		return &netxlite.ErrWrapper{
			Failure:    netxlite.FailureHTTPInvalidRedirectLocationHost,
			Operation:  netxlite.HTTPRoundTripOperation,
			WrappedErr: nil,
		}
	}
	return nil
}
