package dslx

//
// TLS adapters for HTTP
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPRequestOverTLS returns a Func that issues HTTP requests over TLS.
func HTTPRequestOverTLS(rt Runtime, options ...HTTPRequestOption) Func[*TLSConnection, *Maybe[*HTTPResponse]] {
	return Compose2(HTTPConnectionTLS(rt), HTTPRequest(rt, options...))
}

// HTTPConnectionTLS converts a TLS connection into an HTTP connection.
func HTTPConnectionTLS(rt Runtime) Func[*TLSConnection, *Maybe[*HTTPConnection]] {
	return StageAdapter[*TLSConnection, *HTTPConnection](func(ctx context.Context, input *TLSConnection) *Maybe[*HTTPConnection] {
		// TODO(https://github.com/ooni/probe/issues/2534): here we're using the QUIRKY netxlite.NewHTTPTransport
		// function, but we can probably avoid using it, given that this code is
		// not using tracing and does not care about those quirks.
		httpTransport := netxlite.NewHTTPTransport(
			rt.Logger(),
			netxlite.NewNullDialer(),
			netxlite.NewSingleUseTLSDialer(input.Conn),
		)
		state := &HTTPConnection{
			Address:               input.Address,
			Domain:                input.Domain,
			Network:               input.Network,
			Scheme:                "https",
			TLSNegotiatedProtocol: input.TLSState.NegotiatedProtocol,
			Trace:                 input.Trace,
			Transport:             httpTransport,
		}
		return &Maybe[*HTTPConnection]{
			Error:        nil,
			Observations: nil,
			Operation:    "", // we cannot fail, so no need to store operation name
			State:        state,
		}
	})
}
