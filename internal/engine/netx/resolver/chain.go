package resolver

import (
	"context"
)

// ChainResolver is a chain resolver. The primary resolver is used first and, if that
// fails, we then attempt with the secondary resolver.
type ChainResolver struct {
	Primary   Resolver
	Secondary Resolver
}

// LookupHost implements Resolver.LookupHost
func (c ChainResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := c.Primary.LookupHost(ctx, hostname)
	if err != nil {
		addrs, err = c.Secondary.LookupHost(ctx, hostname)
	}
	return addrs, err
}

// Network implements Resolver.Network
func (c ChainResolver) Network() string {
	return "chain"
}

// Address implements Resolver.Address
func (c ChainResolver) Address() string {
	return ""
}

var _ Resolver = ChainResolver{}
