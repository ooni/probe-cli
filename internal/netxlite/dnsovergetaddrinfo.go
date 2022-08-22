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
	testableTimeout   time.Duration
	testableLookupANY func(ctx context.Context, domain string) ([]string, string, error)
}

var _ model.DNSTransport = &dnsOverGetaddrinfoTransport{}

func (txp *dnsOverGetaddrinfoTransport) RoundTrip(
	ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
	if query.Type() != dns.TypeANY {
		return nil, ErrNoDNSTransport
	}
	addrs, _, err := txp.lookup(ctx, query.Domain())
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

type dnsOverGetaddrinfoAddrsAndCNAME struct {
	addrs []string
	cname string
}

func (txp *dnsOverGetaddrinfoTransport) lookup(
	ctx context.Context, hostname string) ([]string, string, error) {
	// This code forces adding a shorter timeout to the domain name
	// resolutions when using the system resolver. We have seen cases
	// in which such a timeout becomes too large. One such case is
	// described in https://github.com/ooni/probe/issues/1726.
	addrsch, errch := make(chan *dnsOverGetaddrinfoAddrsAndCNAME, 1), make(chan error, 1)
	ctx, cancel := context.WithTimeout(ctx, txp.timeout())
	defer cancel()
	go func() {
		addrs, cname, err := txp.lookupfn()(ctx, hostname)
		if err != nil {
			errch <- err
			return
		}
		addrsch <- &dnsOverGetaddrinfoAddrsAndCNAME{
			addrs: addrs,
			cname: cname,
		}
	}()
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case p := <-addrsch:
		return p.addrs, p.cname, nil
	case err := <-errch:
		return nil, "", err
	}
}

func (txp *dnsOverGetaddrinfoTransport) timeout() time.Duration {
	if txp.testableTimeout > 0 {
		return txp.testableTimeout
	}
	return 15 * time.Second
}

func (txp *dnsOverGetaddrinfoTransport) lookupfn() func(ctx context.Context, domain string) ([]string, string, error) {
	if txp.testableLookupANY != nil {
		return txp.testableLookupANY
	}
	return getaddrinfoLookupANY
}

func (txp *dnsOverGetaddrinfoTransport) RequiresPadding() bool {
	return false
}

func (txp *dnsOverGetaddrinfoTransport) Network() string {
	return getaddrinfoResolverNetwork()
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
