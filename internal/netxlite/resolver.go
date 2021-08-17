package netxlite

import (
	"context"
	"net"
	"time"

	"golang.org/x/net/idna"
)

// Resolver performs domain name resolutions.
type Resolver interface {
	// LookupHost behaves like net.Resolver.LookupHost.
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)
}

// ResolverSystem is the system resolver.
type ResolverSystem struct{}

var _ Resolver = &ResolverSystem{}

// LookupHost implements Resolver.LookupHost.
func (r *ResolverSystem) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, hostname)
}

// Network implements Resolver.Network.
func (r *ResolverSystem) Network() string {
	return "system"
}

// Address implements Resolver.Address.
func (r *ResolverSystem) Address() string {
	return ""
}

// DefaultResolver is the resolver we use by default.
var DefaultResolver = &ResolverSystem{}

// ResolverLogger is a resolver that emits events
type ResolverLogger struct {
	Resolver
	Logger Logger
}

var _ Resolver = &ResolverLogger{}

// LookupHost returns the IP addresses of a host
func (r *ResolverLogger) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	r.Logger.Debugf("resolve %s...", hostname)
	start := time.Now()
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	elapsed := time.Since(start)
	if err != nil {
		r.Logger.Debugf("resolve %s... %s in %s", hostname, err, elapsed)
		return nil, err
	}
	r.Logger.Debugf("resolve %s... %+v in %s", hostname, addrs, elapsed)
	return addrs, nil
}

type resolverNetworker interface {
	Network() string
}

// Network implements Resolver.Network.
func (r *ResolverLogger) Network() string {
	if rn, ok := r.Resolver.(resolverNetworker); ok {
		return rn.Network()
	}
	return "logger"
}

type resolverAddresser interface {
	Address() string
}

// Address implements Resolver.Address.
func (r *ResolverLogger) Address() string {
	if ra, ok := r.Resolver.(resolverAddresser); ok {
		return ra.Address()
	}
	return ""
}

// ResolverIDNA supports resolving Internationalized Domain Names.
//
// See RFC3492 for more information.
type ResolverIDNA struct {
	Resolver
}

// LookupHost implements Resolver.LookupHost.
func (r *ResolverIDNA) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	host, err := idna.ToASCII(hostname)
	if err != nil {
		return nil, err
	}
	return r.Resolver.LookupHost(ctx, host)
}

// Network implements Resolver.Network.
func (r *ResolverIDNA) Network() string {
	if rn, ok := r.Resolver.(resolverNetworker); ok {
		return rn.Network()
	}
	return "idna"
}

// Address implements Resolver.Address.
func (r *ResolverIDNA) Address() string {
	if ra, ok := r.Resolver.(resolverAddresser); ok {
		return ra.Address()
	}
	return ""
}
