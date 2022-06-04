package netxlite

//
// Parallel DNS resolver implementation
//

import (
	"context"
	"net"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// ParallelResolver uses a transport and performs a LookupHost
// operation in a parallel fashion, hence its name.
//
// You should probably use NewUnwrappedParallelResolver to
// create a new instance of this type.
type ParallelResolver struct {
	// Txp is the MANDATORY underlying DNS transport.
	Txp model.DNSTransport
}

var _ model.Resolver = &ParallelResolver{}

// UnwrappedParallelResolver creates a new ParallelResolver instance. This instance is
// not wrapped and you should wrap if before using it.
func NewUnwrappedParallelResolver(t model.DNSTransport) *ParallelResolver {
	return &ParallelResolver{
		Txp: t,
	}
}

// Transport returns the transport being used.
func (r *ParallelResolver) Transport() model.DNSTransport {
	return r.Txp
}

// Network returns the "network" of the underlying transport.
func (r *ParallelResolver) Network() string {
	return r.Txp.Network()
}

// Address returns the "address" of the underlying transport.
func (r *ParallelResolver) Address() string {
	return r.Txp.Address()
}

// CloseIdleConnections closes idle connections, if any.
func (r *ParallelResolver) CloseIdleConnections() {
	r.Txp.CloseIdleConnections()
}

// LookupHost performs an A lookup in parallel with an AAAA lookup.
func (r *ParallelResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	ach := make(chan *parallelResolverResult)
	go r.lookupHost(ctx, hostname, dns.TypeA, ach)
	aaaach := make(chan *parallelResolverResult)
	go r.lookupHost(ctx, hostname, dns.TypeAAAA, aaaach)
	ares := <-ach
	aaaares := <-aaaach
	if ares.err != nil && aaaares.err != nil {
		// Note: we choose to return the A error because we assume that
		// it's the more meaningful one: the AAAA error may just be telling
		// us that there is no AAAA record for the website.
		return nil, ares.err
	}
	var addrs []string
	addrs = append(addrs, ares.addrs...)
	addrs = append(addrs, aaaares.addrs...)
	if len(addrs) < 1 {
		return nil, ErrOODNSNoAnswer
	}
	return addrs, nil
}

// LookupHTTPS implements Resolver.LookupHTTPS.
func (r *ParallelResolver) LookupHTTPS(
	ctx context.Context, hostname string) (*model.HTTPSSvc, error) {
	encoder := &DNSEncoderMiekg{}
	query := encoder.Encode(hostname, dns.TypeHTTPS, r.Txp.RequiresPadding())
	response, err := r.Txp.RoundTrip(ctx, query)
	if err != nil {
		return nil, err
	}
	return response.DecodeHTTPS()
}

// parallelResolverResult is the internal representation of a
// lookup using either the A or the AAAA query type.
type parallelResolverResult struct {
	addrs []string
	err   error
}

// lookupHost issues a lookup host query for the specified qtype (e.g., dns.A).
func (r *ParallelResolver) lookupHost(ctx context.Context, hostname string,
	qtype uint16, out chan<- *parallelResolverResult) {
	encoder := &DNSEncoderMiekg{}
	query := encoder.Encode(hostname, qtype, r.Txp.RequiresPadding())
	response, err := r.Txp.RoundTrip(ctx, query)
	if err != nil {
		out <- &parallelResolverResult{
			addrs: []string{},
			err:   err,
		}
		return
	}
	addrs, err := response.DecodeLookupHost()
	out <- &parallelResolverResult{
		addrs: addrs,
		err:   err,
	}
}

// LookupNS implements Resolver.LookupNS.
func (r *ParallelResolver) LookupNS(
	ctx context.Context, hostname string) ([]*net.NS, error) {
	encoder := &DNSEncoderMiekg{}
	query := encoder.Encode(hostname, dns.TypeNS, r.Txp.RequiresPadding())
	response, err := r.Txp.RoundTrip(ctx, query)
	if err != nil {
		return nil, err
	}
	return response.DecodeNS()
}
