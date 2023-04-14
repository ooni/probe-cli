package dslx

//
// QUIC adapters for HTTP (HTTP/3)
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPRequestOverQUIC returns a Func that issues HTTP requests over QUIC.
func HTTPRequestOverQUIC(options ...HTTPRequestOption) Func[*QUICConnection, *Maybe[*HTTPResponse]] {
	return Compose2(HTTPTransportQUIC(), HTTPRequest(options...))
}

// HTTPTransportQUIC converts a QUIC connection into an HTTP transport.
func HTTPTransportQUIC() Func[*QUICConnection, *Maybe[*HTTPTransport]] {
	return &httpTransportQUICFunc{}
}

// httpTransportQUICFunc is the function returned by HTTPTransportQUIC.
type httpTransportQUICFunc struct{}

// Apply implements Func.
func (f *httpTransportQUICFunc) Apply(
	ctx context.Context, input *QUICConnection) *Maybe[*HTTPTransport] {
	// create transport
	httpTransport := netxlite.NewHTTP3Transport(
		input.Logger,
		netxlite.NewSingleUseQUICDialer(input.QUICConn),
		input.TLSConfig,
	)

	state := &HTTPTransport{
		Address:               input.Address,
		Domain:                input.Domain,
		IDGenerator:           input.IDGenerator,
		Logger:                input.Logger,
		Network:               input.Network,
		Scheme:                "https",
		TLSNegotiatedProtocol: input.TLSState.NegotiatedProtocol,
		Trace:                 input.Trace,
		Transport:             httpTransport,
		ZeroTime:              input.ZeroTime,
	}
	return &Maybe[*HTTPTransport]{
		Error:        nil,
		Observations: nil,
		Operation:    "", // we cannot fail, so no need to store operation name
		State:        state,
	}
}
