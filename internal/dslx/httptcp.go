package dslx

//
// TCP adapters for HTTP
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPRequestOverTCP returns a Func that issues HTTP requests over TCP.
func HTTPRequestOverTCP(rt Runtime, options ...HTTPRequestOption) Stage[*TCPConnection, *Maybe[*HTTPResponse]] {
	return Compose2(HTTPConnectionTCP(rt), HTTPRequest(rt, options...))
}

// HTTPConnectionTCP converts a TCP connection into an HTTP connection.
func HTTPConnectionTCP(rt Runtime) Stage[*TCPConnection, *Maybe[*HTTPConnection]] {
	return StageAdapter[*TCPConnection, *HTTPConnection](func(ctx context.Context, input *TCPConnection) *Maybe[*HTTPConnection] {
		// TODO(https://github.com/ooni/probe/issues/2534): here we're using the QUIRKY netxlite.NewHTTPTransport
		// function, but we can probably avoid using it, given that this code is
		// not using tracing and does not care about those quirks.
		httpTransport := netxlite.NewHTTPTransport(
			rt.Logger(),
			netxlite.NewSingleUseDialer(input.Conn),
			netxlite.NewNullTLSDialer(),
		)
		state := &HTTPConnection{
			Address:               input.Address,
			Domain:                input.Domain,
			Network:               input.Network,
			Scheme:                "http",
			TLSNegotiatedProtocol: "",
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
