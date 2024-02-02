package bytecounter

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// WrapWithContextAwareSystemResolver wraps the given resolver with a resolver that
// is aware of context-byte counting. See MaybeWrapSystemResolver for a list of caveats.
func WrapWithContextAwareSystemResolver(reso model.Resolver) model.Resolver {
	return &ContextAwareSystemResolver{reso}
}

// ContextAwareSystemResolver is a [model.Resolver] that knows how to count bytes
// sent and received. We typically use this for the system resolver only because for
// other resolvers we are better off just wrapping their connections.
type ContextAwareSystemResolver struct {
	R model.Resolver
}

// Address implements model.Resolver.
func (r *ContextAwareSystemResolver) Address() string {
	return r.R.Address()
}

// CloseIdleConnections implements model.Resolver.
func (r *ContextAwareSystemResolver) CloseIdleConnections() {
	r.R.CloseIdleConnections()
}

func (r *ContextAwareSystemResolver) wrap(ctx context.Context) model.Resolver {
	return MaybeWrapSystemResolver(MaybeWrapSystemResolver(
		r.R, ContextSessionByteCounter(ctx)), ContextExperimentByteCounter(ctx))
}

// LookupHTTPS implements model.Resolver.
func (r *ContextAwareSystemResolver) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return r.wrap(ctx).LookupHTTPS(ctx, domain)
}

// LookupHost implements model.Resolver.
func (r *ContextAwareSystemResolver) LookupHost(ctx context.Context, hostname string) (addrs []string, err error) {
	return r.wrap(ctx).LookupHost(ctx, hostname)
}

// LookupNS implements model.Resolver.
func (r *ContextAwareSystemResolver) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	return r.wrap(ctx).LookupNS(ctx, domain)
}

// Network implements model.Resolver.
func (r *ContextAwareSystemResolver) Network() string {
	return r.R.Network()
}

// MaybeWrapSystemResolver takes in input a Resolver and either wraps it
// to perform byte counting, if this counter is not nil, or just returns to the
// caller the original resolver, when the counter is nil.
//
// # Caveat
//
// The returned resolver will only approximately estimate the bytes
// sent and received by this resolver if this resolver is the system
// resolver. For more accurate counting when using DNS over HTTPS,
// you should instead count at the HTTP transport level. If you are
// using DNS over TCP, DNS over TLS, or DNS over UDP, you should instead
// count the bytes by just wrapping the connections you're using.
func MaybeWrapSystemResolver(reso model.Resolver, counter *Counter) model.Resolver {
	if counter != nil {
		reso = WrapSystemResolver(reso, counter)
	}
	return reso
}

// WrapSystemResolver creates a new byte-counting-aware resolver. This function
// returns a resolver with the same bugs of [MaybeWrapSystemResolver].
func WrapSystemResolver(reso model.Resolver, counter *Counter) model.Resolver {
	return &resolver{
		Resolver: reso,
		Counter:  counter,
	}
}

// resolver is the type returned by WrapResolver.
type resolver struct {
	Resolver model.Resolver
	Counter  *Counter
}

// Address implements model.Resolver
func (r *resolver) Address() string {
	return r.Resolver.Address()
}

// CloseIdleConnections implements model.Resolver
func (r *resolver) CloseIdleConnections() {
	r.Resolver.CloseIdleConnections()
}

// LookupHTTPS implements model.Resolver
func (r *resolver) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	r.updateCounterBytesSent(domain, 1)
	out, err := r.Resolver.LookupHTTPS(ctx, domain)
	r.updateCounterBytesRecv(err)
	return out, err
}

// LookupHost implements model.Resolver
func (r *resolver) LookupHost(ctx context.Context, hostname string) (addrs []string, err error) {
	r.updateCounterBytesSent(hostname, 2)
	out, err := r.Resolver.LookupHost(ctx, hostname)
	r.updateCounterBytesRecv(err)
	return out, err
}

// LookupNS implements model.Resolver
func (r *resolver) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	r.updateCounterBytesSent(domain, 1)
	out, err := r.Resolver.LookupNS(ctx, domain)
	r.updateCounterBytesRecv(err)
	return out, err
}

// Network implements model.Resolver
func (r *resolver) Network() string {
	return r.Resolver.Network()
}

// updateCounterBytesSent estimates the bytes sent.
func (r *resolver) updateCounterBytesSent(domain string, n int) {
	// Assume we are sending N queries for the given domain, which is the
	// correct byte counting strategy when using the system resolver
	r.Counter.Sent.Add(int64(n * len(domain)))
}

// updateCounterBytesRecv estimates the bytes recv.
func (r *resolver) updateCounterBytesRecv(err error) {
	if err != nil {
		switch err.Error() {
		case netxlite.FailureDNSNXDOMAINError,
			netxlite.FailureDNSNoAnswer,
			netxlite.FailureDNSRefusedError,
			netxlite.FailureDNSNonRecoverableFailure,
			netxlite.FailureDNSServfailError:
			// In case it seems we received a message, let us
			// pretend overall it was 128 bytes
			r.Counter.Received.Add(128)
		default:
			// In this case we assume we did not receive any byte
		}
		return
	}
	// On success we assume we received 256 bytes
	r.Counter.Received.Add(256)
}
