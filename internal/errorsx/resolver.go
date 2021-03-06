package errorsx

import (
	"context"
	"errors"
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
		Classifier: classifyResolveFailure,
		Error:      err,
		Operation:  ResolveOperation,
	}.MaybeBuild()
	return addrs, err
}

// classifyResolveFailure is a classifier to translate DNS resolving errors to OONI error strings.
func classifyResolveFailure(err error) string {
	if errors.Is(err, ErrDNSBogon) {
		return FailureDNSBogonError // not in MK
	}
	return toFailureString(err)
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
