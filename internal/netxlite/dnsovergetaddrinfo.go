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
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// dnsOverGetaddrinfoTransport is a DNSTransport using getaddrinfo.
type dnsOverGetaddrinfoTransport struct {
	// (OPTIONAL) allows to run tests with a short timeout
	testableTimeout time.Duration

	// (OPTIONAL) allows to mock the underlying getaddrinfo call
	testableLookupANY func(ctx context.Context, domain string) ([]string, string, error)
}

// NewDNSOverGetaddrinfoTransport creates a new dns-over-getaddrinfo transport.
func NewDNSOverGetaddrinfoTransport() model.DNSTransport {
	return &dnsOverGetaddrinfoTransport{}
}

var _ model.DNSTransport = &dnsOverGetaddrinfoTransport{}

func (txp *dnsOverGetaddrinfoTransport) RoundTrip(
	ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
	if query.Type() != dns.TypeANY {
		return nil, ErrNoDNSTransport
	}
	addrs, cname, err := txp.lookup(ctx, query.Domain())
	if err != nil {
		return nil, err
	}
	resp := &dnsOverGetaddrinfoResponse{
		addrs: addrs,
		cname: cname,
		query: query,
	}
	return resp, nil
}

type dnsOverGetaddrinfoResponse struct {
	addrs []string
	cname string
	query model.DNSQuery
}

// Used to move addrs and cname out of the worker goroutine
type dnsOverGetaddrinfoAddrsAndCNAME struct {
	// List of resolved addresses (it's a bug if this is empty)
	addrs []string

	// Resolved CNAME or empty string
	cname string
}

func (txp *dnsOverGetaddrinfoTransport) lookup(
	ctx context.Context, hostname string) ([]string, string, error) {
	// This code forces adding a shorter timeout to the domain name
	// resolutions when using the system resolver. We have seen cases
	// in which such a timeout becomes too large. One such case is
	// described in https://github.com/ooni/probe/issues/1726.
	outch, errch := make(chan *dnsOverGetaddrinfoAddrsAndCNAME, 1), make(chan error, 1)
	ctx, cancel := context.WithTimeout(ctx, txp.timeout())
	defer cancel()
	go func() {
		addrs, cname, err := txp.lookupfn()(ctx, hostname)
		if err != nil {
			errch <- err
			return
		}
		outch <- &dnsOverGetaddrinfoAddrsAndCNAME{
			addrs: addrs,
			cname: cname,
		}
	}()
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case p := <-outch:
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
	return TProxy.GetaddrinfoLookupANY
}

func (txp *dnsOverGetaddrinfoTransport) RequiresPadding() bool {
	return false
}

func (txp *dnsOverGetaddrinfoTransport) Network() string {
	return TProxy.GetaddrinfoResolverNetwork()
}

func (txp *dnsOverGetaddrinfoTransport) Address() string {
	return ""
}

func (txp *dnsOverGetaddrinfoTransport) CloseIdleConnections() {
	// nothing
}

func (r *dnsOverGetaddrinfoResponse) Query() model.DNSQuery {
	runtimex.PanicIfNil(r.query, "dnsOverGetaddrinfoResponse with nil query")
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
	if len(r.addrs) <= 0 {
		return nil, ErrOODNSNoAnswer
	}
	return r.addrs, nil
}

func (r *dnsOverGetaddrinfoResponse) DecodeNS() ([]*net.NS, error) {
	return nil, ErrNoDNSTransport
}

func (r *dnsOverGetaddrinfoResponse) DecodeCNAME() (string, error) {
	if r.cname == "" {
		return "", ErrOODNSNoAnswer
	}
	return r.cname, nil
}
