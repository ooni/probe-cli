package dslx

//
// QUIC adapters for HTTP (HTTP/3)
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

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
		Skipped:      false,
		State:        state,
	}
}
