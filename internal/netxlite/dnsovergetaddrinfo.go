package netxlite

//
// DNS over getaddrinfo: fake transport to allow us to observe
// lookups using getaddrinfo as a DNSTransport.
//

import (
	"context"
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// dnsOverGetaddrinfoTransport is a DNSTransport using getaddrinfo.
type dnsOverGetaddrinfoTransport struct {
	testableTimeout    time.Duration
	testableLookupHost func(ctx context.Context, domain string) ([]string, error)
}

var _ model.DNSTransport = &dnsOverGetaddrinfoTransport{}

func (txp *dnsOverGetaddrinfoTransport) RoundTrip(
	ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
	if query.Type() != dns.TypeANY {
		return nil, ErrNoDNSTransport
	}
	addrs, err := txp.lookup(ctx, query.Domain())
	if err != nil {
		return nil, err
	}
	resp := &dnsOverGetaddrinfoResponse{
		addrs: addrs,
		query: query,
	}
	return resp, nil
}

type dnsOverGetaddrinfoResponse struct {
	addrs []string
	query model.DNSQuery
}

func (txp *dnsOverGetaddrinfoTransport) lookup(
	ctx context.Context, hostname string) ([]string, error) {
	// This code forces adding a shorter timeout to the domain name
	// resolutions when using the system resolver. We have seen cases
	// in which such a timeout becomes too large. One such case is
	// described in https://github.com/ooni/probe/issues/1726.
	addrsch, errch := make(chan []string, 1), make(chan error, 1)
	ctx, cancel := context.WithTimeout(ctx, txp.timeout())
	defer cancel()
	go func() {
		addrs, err := txp.lookupfn()(ctx, hostname)
		if err != nil {
			errch <- err
			return
		}
		addrsch <- addrs
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case addrs := <-addrsch:
		return addrs, nil
	case err := <-errch:
		return nil, err
	}
}

func (txp *dnsOverGetaddrinfoTransport) timeout() time.Duration {
	if txp.testableTimeout > 0 {
		return txp.testableTimeout
	}
	return 15 * time.Second
}

func (txp *dnsOverGetaddrinfoTransport) lookupfn() func(ctx context.Context, domain string) ([]string, error) {
	if txp.testableLookupHost != nil {
		return txp.testableLookupHost
	}
	return TProxy.DefaultResolver().LookupHost
}

func (txp *dnsOverGetaddrinfoTransport) RequiresPadding() bool {
	return false
}

func (txp *dnsOverGetaddrinfoTransport) Network() string {
	return TProxy.DefaultResolver().Network()
}

func (txp *dnsOverGetaddrinfoTransport) Address() string {
	return ""
}

func (txp *dnsOverGetaddrinfoTransport) CloseIdleConnections() {
	// nothing
}

func (r *dnsOverGetaddrinfoResponse) Query() model.DNSQuery {
	return r.query
}

func (r *dnsOverGetaddrinfoResponse) Bytes() []byte {
	return nil
}

func (r *dnsOverGetaddrinfoResponse) Rcode() int {
	return 0
}

func (r *dnsOverGetaddrinfoResponse) DecodeHTTPS() (*model.HTTPSSvc, error) {
	return nil, ErrNoDNSTransport
}

func (r *dnsOverGetaddrinfoResponse) DecodeLookupHost() ([]string, error) {
	return r.addrs, nil
}

func (r *dnsOverGetaddrinfoResponse) DecodeNS() ([]*net.NS, error) {
	return nil, ErrNoDNSTransport
}
