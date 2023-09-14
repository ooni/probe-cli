package dslx

//
// TCP adapters for HTTP
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPRequestOverTCP returns a Func that issues HTTP requests over TCP.
func HTTPRequestOverTCP(options ...HTTPRequestOption) Func[*TCPConnection, *Maybe[*HTTPResponse]] {
	return Compose2(HTTPTransportTCP(), HTTPRequest(options...))
}

// HTTPTransportTCP converts a TCP connection into an HTTP transport.
func HTTPTransportTCP() Func[*TCPConnection, *Maybe[*HTTPTransport]] {
	return &httpTransportTCPFunc{}
}

// httpTransportTCPFunc is the function returned by HTTPTransportTCP
type httpTransportTCPFunc struct{}

// Apply implements Func
func (f *httpTransportTCPFunc) Apply(
	ctx context.Context, input *TCPConnection) *Maybe[*HTTPTransport] {
	// TODO(https://github.com/ooni/probe/issues/2534): here we're using the QUIRKY netxlite.NewHTTPTransport
	// function, but we can probably avoid using it, given that this code is
	// not using tracing and does not care about those quirks.
	httpTransport := netxlite.NewHTTPTransport(
		input.Logger,
		netxlite.NewSingleUseDialer(input.Conn),
		netxlite.NewNullTLSDialer(),
	)
	state := &HTTPTransport{
		Address:               input.Address,
		Domain:                input.Domain,
		IDGenerator:           input.IDGenerator,
		Logger:                input.Logger,
		Network:               input.Network,
		Scheme:                "http",
		TLSNegotiatedProtocol: "",
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
