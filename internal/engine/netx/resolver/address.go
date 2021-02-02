package resolver

import (
	"context"
	"net"
)

// AddressResolver is a resolver that knows how to correctly
// resolve IP addresses to themselves.
type AddressResolver struct {
	Resolver
}

// LookupHost implements Resolver.LookupHost
func (r AddressResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	if net.ParseIP(hostname) != nil {
		return []string{hostname}, nil
	}
	return r.Resolver.LookupHost(ctx, hostname)
}

var _ Resolver = AddressResolver{}
