package dnsx

import (
	"context"
	"errors"
	"net"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

// SerialResolver is a resolver that first issues an A query and then
// issues an AAAA query for the requested domain.
type SerialResolver struct {
	Encoder     Encoder
	Decoder     Decoder
	NumTimeouts *atomicx.Int64
	Txp         RoundTripper
}

// NewSerialResolver creates a new OONI Resolver instance.
func NewSerialResolver(t RoundTripper) *SerialResolver {
	return &SerialResolver{
		Encoder:     &MiekgEncoder{},
		Decoder:     &MiekgDecoder{},
		NumTimeouts: &atomicx.Int64{},
		Txp:         t,
	}
}

// Transport returns the transport being used.
func (r *SerialResolver) Transport() RoundTripper {
	return r.Txp
}

// Network implements Resolver.Network
func (r *SerialResolver) Network() string {
	return r.Txp.Network()
}

// Address implements Resolver.Address
func (r *SerialResolver) Address() string {
	return r.Txp.Address()
}

// CloseIdleConnections closes idle connections.
func (r *SerialResolver) CloseIdleConnections() {
	r.Txp.CloseIdleConnections()
}

// LookupHost implements Resolver.LookupHost.
func (r *SerialResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	var addrs []string
	addrsA, errA := r.lookupHostWithRetry(ctx, hostname, dns.TypeA)
	addrsAAAA, errAAAA := r.lookupHostWithRetry(ctx, hostname, dns.TypeAAAA)
	if errA != nil && errAAAA != nil {
		return nil, errA
	}
	addrs = append(addrs, addrsA...)
	addrs = append(addrs, addrsAAAA...)
	return addrs, nil
}

func (r *SerialResolver) lookupHostWithRetry(
	ctx context.Context, hostname string, qtype uint16) ([]string, error) {
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
	// bugfix: we MUST return one of the errors otherwise we confuse the
	// mechanism in errwrap that classifies the root cause operation, since
	// it would not be able to find a child with a major operation error
	return nil, errorslist[0]
}

// lookupHostWithoutRetry issues a lookup host query for the specified
// qtype (dns.A or dns.AAAA) without retrying on failure.
func (r *SerialResolver) lookupHostWithoutRetry(
	ctx context.Context, hostname string, qtype uint16) ([]string, error) {
	querydata, err := r.Encoder.Encode(hostname, qtype, r.Txp.RequiresPadding())
	if err != nil {
		return nil, err
	}
	replydata, err := r.Txp.RoundTrip(ctx, querydata)
	if err != nil {
		return nil, err
	}
	return r.Decoder.DecodeLookupHost(qtype, replydata)
}
