package netxlite

//
// Generic DNS transport code.
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// wrapDNSTransport wraps a DNSTransport to provide error wrapping.
func wrapDNSTransport(txp model.DNSTransport) (out model.DNSTransport) {
	out = &dnsTransportErrWrapper{
		DNSTransport: txp,
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
		return nil, NewErrWrapper(ClassifyResolverError, DNSRoundTripOperation, err)
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
