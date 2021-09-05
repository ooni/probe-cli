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

// resolverSystem is the system resolver.
type resolverSystem struct{}

var _ Resolver = &resolverSystem{}

// LookupHost implements Resolver.LookupHost.
func (r *resolverSystem) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, hostname)
}

// Network implements Resolver.Network.
func (r *resolverSystem) Network() string {
	return "system"
}

// Address implements Resolver.Address.
func (r *resolverSystem) Address() string {
	return ""
}

// DefaultResolver is the resolver we use by default.
var DefaultResolver = &resolverSystem{}

// resolverLogger is a resolver that emits events
type resolverLogger struct {
	Resolver
	Logger Logger
}

var _ Resolver = &resolverLogger{}

// LookupHost returns the IP addresses of a host
func (r *resolverLogger) LookupHost(ctx context.Context, hostname string) ([]string, error) {
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
func (r *resolverLogger) Network() string {
	if rn, ok := r.Resolver.(resolverNetworker); ok {
		return rn.Network()
	}
	return "logger"
}

type resolverAddresser interface {
	Address() string
}

// Address implements Resolver.Address.
func (r *resolverLogger) Address() string {
	if ra, ok := r.Resolver.(resolverAddresser); ok {
		return ra.Address()
	}
	return ""
}

// resolverIDNA supports resolving Internationalized Domain Names.
//
// See RFC3492 for more information.
type resolverIDNA struct {
	Resolver
}

// LookupHost implements Resolver.LookupHost.
func (r *resolverIDNA) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	host, err := idna.ToASCII(hostname)
	if err != nil {
		return nil, err
	}
	return r.Resolver.LookupHost(ctx, host)
}

// Network implements Resolver.Network.
func (r *resolverIDNA) Network() string {
	if rn, ok := r.Resolver.(resolverNetworker); ok {
		return rn.Network()
	}
	return "idna"
}

// Address implements Resolver.Address.
func (r *resolverIDNA) Address() string {
	if ra, ok := r.Resolver.(resolverAddresser); ok {
		return ra.Address()
	}
	return ""
}
