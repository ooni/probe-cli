package netxlite

//
// Serial DNS resolver implementation
//

import (
	"context"
	"errors"
	"net"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// SerialResolver uses a transport and performs a LookupHost
// operation in a serial fashion (query for A first, wait for response,
// then query for AAAA, and wait for response), hence its name.
//
// You should probably use NewSerialResolver to create a new instance.
//
// Deprecated: please use ParallelResolver in new code. We cannot
// remove this code as long as we use tracing for measuring.
//
// QUIRK: unlike the ParallelResolver, this resolver's LookupHost retries
// each query three times for soft errors.
type SerialResolver struct {
	// Encoder is the MANDATORY encoder to use.
	Encoder model.DNSEncoder

	// Decoder is the MANDATORY decoder to use.
	Decoder model.DNSDecoder

	// NumTimeouts is MANDATORY and counts the number of timeouts.
	NumTimeouts *atomicx.Int64

	// Txp is the MANDATORY underlying DNS transport.
	Txp model.DNSTransport
}

// NewSerialResolver creates a new SerialResolver instance.
func NewSerialResolver(t model.DNSTransport) *SerialResolver {
	return &SerialResolver{
		Encoder:     &DNSEncoderMiekg{},
		Decoder:     &DNSDecoderMiekg{},
		NumTimeouts: &atomicx.Int64{},
		Txp:         t,
	}
}

// Transport returns the transport being used.
func (r *SerialResolver) Transport() model.DNSTransport {
	return r.Txp
}

// Network returns the "network" of the underlying transport.
func (r *SerialResolver) Network() string {
	return r.Txp.Network()
}

// Address returns the "address" of the underlying transport.
func (r *SerialResolver) Address() string {
	return r.Txp.Address()
}

// CloseIdleConnections closes idle connections, if any.
func (r *SerialResolver) CloseIdleConnections() {
	r.Txp.CloseIdleConnections()
}

// LookupHost performs an A lookup followed by an AAAA lookup for hostname.
func (r *SerialResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	var addrs []string
	addrsA, errA := r.lookupHostWithRetry(ctx, hostname, dns.TypeA)
	addrsAAAA, errAAAA := r.lookupHostWithRetry(ctx, hostname, dns.TypeAAAA)
	if errA != nil && errAAAA != nil {
		// Note: we choose to return the errA because we assume that
		// it's the more meaningful one: the errAAAA may just be telling
		// us that there is no AAAA record for the website.
		return nil, errA
	}
	addrs = append(addrs, addrsA...)
	addrs = append(addrs, addrsAAAA...)
	return addrs, nil
}

// LookupHTTPS implements Resolver.LookupHTTPS.
func (r *SerialResolver) LookupHTTPS(
	ctx context.Context, hostname string) (*model.HTTPSSvc, error) {
	query := r.Encoder.Encode(hostname, dns.TypeHTTPS, r.Txp.RequiresPadding())
	response, err := r.Txp.RoundTrip(ctx, query)
	if err != nil {
		return nil, err
	}
	return response.DecodeHTTPS()
}

func (r *SerialResolver) lookupHostWithRetry(
	ctx context.Context, hostname string, qtype uint16) ([]string, error) {
	// QUIRK: retrying has been there since the beginning so we need to
	// keep it as long as we're using tracing for measuring.
	var errorslist []error
	for i := 0; i < 3; i++ {
		replies, err := r.lookupHostWithoutRetry(ctx, hostname, qtype)
		if err == nil {
			return replies, nil
		}
		errorslist = append(errorslist, err)
		var operr *net.OpError
		if !errors.As(err, &operr) || !operr.Timeout() {
			// The first error is the one that is most likely to be caused
			// by the network. Subsequent errors are more likely to be caused
			// by context deadlines. So, the first error is attached to an
			// operation, while subsequent errors may possibly not be. If
			// so, the resulting failing operation is not correct.
			break
		}
		r.NumTimeouts.Add(1)
	}
	// QUIRK: we MUST return one of the errors otherwise we confuse the
	// mechanism in errwrap that classifies the root cause operation, since
	// it would not be able to find a child with a major operation error.
	return nil, errorslist[0]
}

// lookupHostWithoutRetry issues a lookup host query for the specified
// qtype (dns.A or dns.AAAA) without retrying on failure.
func (r *SerialResolver) lookupHostWithoutRetry(
	ctx context.Context, hostname string, qtype uint16) ([]string, error) {
	query := r.Encoder.Encode(hostname, qtype, r.Txp.RequiresPadding())
	response, err := r.Txp.RoundTrip(ctx, query)
	if err != nil {
		return nil, err
	}
	return response.DecodeLookupHost()
}

// LookupNS implements Resolver.LookupNS.
func (r *SerialResolver) LookupNS(
	ctx context.Context, hostname string) ([]*net.NS, error) {
	query := r.Encoder.Encode(hostname, dns.TypeNS, r.Txp.RequiresPadding())
	response, err := r.Txp.RoundTrip(ctx, query)
	if err != nil {
		return nil, err
	}
	return response.DecodeNS()
}
