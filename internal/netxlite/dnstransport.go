package netxlite

//
// Generic DNS transport code.
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// WrapDNSTransport wraps a DNSTransport to provide error wrapping. This function will
// apply all the provided wrappers around the default transport wrapping. If any of the
// wrappers is nil, we just silently and gracefully ignore it.
func WrapDNSTransport(txp model.DNSTransport,
	wrappers ...model.DNSTransportWrapper) (out model.DNSTransport) {
	out = &dnsTransportErrWrapper{
		DNSTransport: txp,
	}
	for _, wrapper := range wrappers {
		if wrapper == nil {
			continue // skip as documented
		}
		out = wrapper.WrapDNSTransport(out) // compose with user-provided wrappers
	}
	return
}

// dnsTransportErrWrapper wraps DNSTransport to provide error wrapping.
type dnsTransportErrWrapper struct {
	DNSTransport model.DNSTransport
}

var _ model.DNSTransport = &dnsTransportErrWrapper{}

func (t *dnsTransportErrWrapper) RoundTrip(
	ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
	resp, err := t.DNSTransport.RoundTrip(ctx, query)
	if err != nil {
		return nil, newErrWrapper(classifyResolverError, DNSRoundTripOperation, err)
	}
	return resp, nil
}

func (t *dnsTransportErrWrapper) RequiresPadding() bool {
	return t.DNSTransport.RequiresPadding()
}

func (t *dnsTransportErrWrapper) Network() string {
	return t.DNSTransport.Network()
}

func (t *dnsTransportErrWrapper) Address() string {
	return t.DNSTransport.Address()
}

func (t *dnsTransportErrWrapper) CloseIdleConnections() {
	t.DNSTransport.CloseIdleConnections()
}
