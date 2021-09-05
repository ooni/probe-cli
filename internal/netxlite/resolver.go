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

	// Network returns the resolver type (e.g., system, dot, doh).
	Network() string

	// Address returns the resolver address (e.g., 8.8.8.8:53).
	Address() string

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()
}

// ResolverConfig contains config for creating a resolver.
type ResolverConfig struct {
	// Logger is the MANDATORY logger to use.
	Logger Logger
}

// NewResolver creates a new resolver.
func NewResolver(config *ResolverConfig) Resolver {
	return &resolverIDNA{
		Resolver: &resolverLogger{
			Resolver: &resolverSystem{},
			Logger:   config.Logger,
		},
	}
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

// CloseIdleConnections implements Resolver.CloseIdleConnections.
func (r *resolverSystem) CloseIdleConnections() {
	// nothing
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
