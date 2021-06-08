package resolver

import (
	"context"
	"net"
)

// SystemResolver is the system resolver.
type SystemResolver struct{}

// LookupHost implements Resolver.LookupHost.
func (r SystemResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, hostname)
}

// Network implements Resolver.Network.
func (r SystemResolver) Network() string {
	return "system"
}

// Address implements Resolver.Address.
func (r SystemResolver) Address() string {
	return ""
}

// Default is the resolver we use by default.
var Default = SystemResolver{}

var _ Resolver = SystemResolver{}
