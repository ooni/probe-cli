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
	return Compose2(HTTPTransportTLS(rt), HTTPRequest(rt, options...))
}

// HTTPTransportTLS converts a TLS connection into an HTTP transport.
func HTTPTransportTLS(rt Runtime) Func[*TLSConnection, *Maybe[*HTTPConnection]] {
	return &httpTransportTLSFunc{rt}
}

// httpTransportTLSFunc is the function returned by HTTPTransportTLS.
type httpTransportTLSFunc struct {
	rt Runtime
}

// Apply implements Func.
func (f *httpTransportTLSFunc) Apply(
	ctx context.Context, input *TLSConnection) *Maybe[*HTTPConnection] {
	// TODO(https://github.com/ooni/probe/issues/2534): here we're using the QUIRKY netxlite.NewHTTPTransport
	// function, but we can probably avoid using it, given that this code is
	// not using tracing and does not care about those quirks.
	httpTransport := netxlite.NewHTTPTransport(
		f.rt.Logger(),
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
}
