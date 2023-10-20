package dslx

//
// TCP adapters for HTTP
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPRequestOverTCP returns a Func that issues HTTP requests over TCP.
func HTTPRequestOverTCP(rt Runtime, options ...HTTPRequestOption) Func[*TCPConnection, *Maybe[*HTTPResponse]] {
	return Compose2(HTTPTransportTCP(rt), HTTPRequest(rt, options...))
}

// HTTPTransportTCP converts a TCP connection into an HTTP transport.
func HTTPTransportTCP(rt Runtime) Func[*TCPConnection, *Maybe[*HTTPTransport]] {
	return &httpTransportTCPFunc{rt}
}

// httpTransportTCPFunc is the function returned by HTTPTransportTCP
type httpTransportTCPFunc struct {
	rt Runtime
}

// Apply implements Func
func (f *httpTransportTCPFunc) Apply(
	ctx context.Context, input *TCPConnection) *Maybe[*HTTPTransport] {
	// TODO(https://github.com/ooni/probe/issues/2534): here we're using the QUIRKY netxlite.NewHTTPTransport
	// function, but we can probably avoid using it, given that this code is
	// not using tracing and does not care about those quirks.
	httpTransport := netxlite.NewHTTPTransport(
		f.rt.Logger(),
		netxlite.NewSingleUseDialer(input.Conn),
		netxlite.NewNullTLSDialer(),
	)
	state := &HTTPTransport{
		Address:               input.Address,
		Domain:                input.Domain,
		Network:               input.Network,
		Scheme:                "http",
		TLSNegotiatedProtocol: "",
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
