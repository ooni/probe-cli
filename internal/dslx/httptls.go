package dslx

//
// TLS adapters for HTTP
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPRequestOverTLS returns a Func that issues HTTP requests over TLS.
func HTTPRequestOverTLS(options ...HTTPRequestOption) Func[*TLSConnection, *Maybe[*HTTPResponse]] {
	return Compose2(HTTPTransportTLS(), HTTPRequest(options...))
}

// HTTPTransportTLS converts a TLS connection into an HTTP transport.
func HTTPTransportTLS() Func[*TLSConnection, *Maybe[*HTTPTransport]] {
	return &httpTransportTLSFunc{}
}

// httpTransportTLSFunc is the function returned by HTTPTransportTLS.
type httpTransportTLSFunc struct{}

// Apply implements Func.
func (f *httpTransportTLSFunc) Apply(
	ctx context.Context, input *TLSConnection) *Maybe[*HTTPTransport] {
	httpTransport := netxlite.NewHTTPTransport(
		input.Logger,
		netxlite.NewNullDialer(),
		netxlite.NewSingleUseTLSDialer(input.Conn),
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
