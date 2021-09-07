package errorsx

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// Resolver is a DNS resolver. The *net.Resolver used by Go implements
// this interface, but other implementations are possible.
type Resolver interface {
	// LookupHost resolves a hostname to a list of IP addresses.
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)
}

// ErrorWrapperResolver is a Resolver that knows about wrapping errors.
type ErrorWrapperResolver struct {
	Resolver
}

var _ Resolver = &ErrorWrapperResolver{}

// LookupHost implements Resolver.LookupHost
func (r *ErrorWrapperResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	err = SafeErrWrapperBuilder{
		Classifier: errorsx.ClassifyResolverError,
		Error:      err,
		Operation:  errorsx.ResolveOperation,
	}.MaybeBuild()
	return addrs, err
}

type resolverNetworker interface {
	Network() string
}

// Network implements Resolver.Network.
func (r *ErrorWrapperResolver) Network() string {
	if rn, ok := r.Resolver.(resolverNetworker); ok {
		return rn.Network()
	}
	return "errorWrapper"
}

type resolverAddresser interface {
	Address() string
}

// Address implements Resolver.Address.
func (r *ErrorWrapperResolver) Address() string {
	if ra, ok := r.Resolver.(resolverAddresser); ok {
		return ra.Address()
	}
	return ""
}
