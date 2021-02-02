package resolver

import (
	"context"

	"golang.org/x/net/idna"
)

// IDNAResolver is to support resolving Internationalized Domain Names.
// See RFC3492 for more information.
type IDNAResolver struct {
	Resolver
}

// LookupHost implements Resolver.LookupHost
func (r IDNAResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	host, err := idna.ToASCII(hostname)
	if err != nil {
		return nil, err
	}
	return r.Resolver.LookupHost(ctx, host)
}

// Network implements Resolver.Network.
func (r IDNAResolver) Network() string {
	return "idna"
}

// Address implements Resolver.Address.
func (r IDNAResolver) Address() string {
	return ""
}

var _ Resolver = IDNAResolver{}
