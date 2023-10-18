package dslx

//
// QUIC adapters for HTTP (HTTP/3)
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPRequestOverQUIC returns a Func that issues HTTP requests over QUIC.
func HTTPRequestOverQUIC(rt Runtime, options ...HTTPRequestOption) Func[*QUICConnection, *Maybe[*HTTPResponse]] {
	return Compose2(HTTPTransportQUIC(rt), HTTPRequest(rt, options...))
}

// HTTPTransportQUIC converts a QUIC connection into an HTTP transport.
func HTTPTransportQUIC(rt Runtime) Func[*QUICConnection, *Maybe[*HTTPTransport]] {
	return &httpTransportQUICFunc{rt}
}

// httpTransportQUICFunc is the function returned by HTTPTransportQUIC.
type httpTransportQUICFunc struct {
	rt Runtime
}

// Apply implements Func.
func (f *httpTransportQUICFunc) Apply(
	ctx context.Context, input *QUICConnection) *Maybe[*HTTPTransport] {
	// create transport
	httpTransport := netxlite.NewHTTP3Transport(
		f.rt.Logger(),
		netxlite.NewSingleUseQUICDialer(input.QUICConn),
		input.TLSConfig,
	)

	state := &HTTPTransport{
		Address:               input.Address,
		Domain:                input.Domain,
		Network:               input.Network,
		Scheme:                "https",
		TLSNegotiatedProtocol: input.TLSState.NegotiatedProtocol,
		Trace:                 input.Trace,
		Transport:             httpTransport,
	}
	return &Maybe[*HTTPTransport]{
		Error:        nil,
		Observations: nil,
		Operation:    "", // we cannot fail, so no need to store operation name
		State:        state,
	}
}
