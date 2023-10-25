package dslx

//
// QUIC adapters for HTTP (HTTP/3)
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPRequestOverQUIC returns a Func that issues HTTP requests over QUIC.
func HTTPRequestOverQUIC(rt Runtime, options ...HTTPRequestOption) Func[*QUICConnection, *HTTPResponse] {
	return Compose2(HTTPConnectionQUIC(rt), HTTPRequest(rt, options...))
}

// HTTPConnectionQUIC converts a QUIC connection into an HTTP connection.
func HTTPConnectionQUIC(rt Runtime) Func[*QUICConnection, *HTTPConnection] {
	return Operation[*QUICConnection, *HTTPConnection](func(ctx context.Context, input *QUICConnection) *Maybe[*HTTPConnection] {
		// create transport
		httpTransport := netxlite.NewHTTP3Transport(
			rt.Logger(),
			netxlite.NewSingleUseQUICDialer(input.QUICConn),
			input.TLSConfig,
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
			Error: nil,
			State: state,
		}
	})
}
