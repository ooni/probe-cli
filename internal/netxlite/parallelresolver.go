package netxlite

//
// Parallel resolver implementation
//

import (
	"context"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// ParallelResolver uses a transport and performs a LookupHost
// operation in a parallel fashion, hence its name.
//
// You should probably use NewUnwrappedParallelResolver to
// create a new instance of this type.
type ParallelResolver struct {
	// Encoder is the MANDATORY encoder to use.
	Encoder model.DNSEncoder

	// Decoder is the MANDATORY decoder to use.
	Decoder model.DNSDecoder

	// NumTimeouts is MANDATORY and counts the number of timeouts.
	NumTimeouts *atomicx.Int64

	// Txp is the MANDATORY underlying DNS transport.
	Txp model.DNSTransport
}

// UnwrappedParallelResolver creates a new ParallelResolver instance. This instance is
// not wrapped and you should wrap if before using it.
func NewUnwrappedParallelResolver(t model.DNSTransport) *ParallelResolver {
	return &ParallelResolver{
		Encoder:     &DNSEncoderMiekg{},
		Decoder:     &DNSDecoderMiekg{},
		NumTimeouts: &atomicx.Int64{},
		Txp:         t,
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
	return addrs, nil
}

// LookupHTTPS implements Resolver.LookupHTTPS.
func (r *ParallelResolver) LookupHTTPS(
	ctx context.Context, hostname string) (*model.HTTPSSvc, error) {
	querydata, err := r.Encoder.Encode(
		hostname, dns.TypeHTTPS, r.Txp.RequiresPadding())
	if err != nil {
		return nil, err
	}
	replydata, err := r.Txp.RoundTrip(ctx, querydata)
	if err != nil {
		return nil, err
	}
	return r.Decoder.DecodeHTTPS(replydata)
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
	querydata, err := r.Encoder.Encode(hostname, qtype, r.Txp.RequiresPadding())
	if err != nil {
		out <- &parallelResolverResult{
			addrs: []string{},
			err:   err,
		}
		return
	}
	replydata, err := r.Txp.RoundTrip(ctx, querydata)
	if err != nil {
		out <- &parallelResolverResult{
			addrs: []string{},
			err:   err,
		}
		return
	}
	addrs, err := r.Decoder.DecodeLookupHost(qtype, replydata)
	out <- &parallelResolverResult{
		addrs: addrs,
		err:   err,
	}
}
