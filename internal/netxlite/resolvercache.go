package netxlite

import (
	"context"
	"net"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// MaybeWrapWithCachingResolver wraps the provided resolver with a resolver
// that remembers the result of previous successful resolutions, if the enabled
// argument is true. Otherwise, we return the unmodified provided resolver.
//
// Bug: the returned resolver only applies caching to LookupHost and any other
// lookup operation returns ErrNoDNSTransport to the caller.
func MaybeWrapWithCachingResolver(enabled bool, reso model.Resolver) model.Resolver {
	if enabled {
		reso = &cacheResolver{
			cache:    map[string][]string{},
			mu:       sync.Mutex{},
			readOnly: false,
			resolver: reso,
		}
	}
	return reso
}

// MaybeWrapWithStaticDNSCache wraps the provided resolver with a resolver that
// checks the given cache before issuing queries to the underlying DNS resolver.
//
// Bug: the returned resolver only applies caching to LookupHost and any other
// lookup operation returns ErrNoDNSTransport to the caller.
func MaybeWrapWithStaticDNSCache(cache map[string][]string, reso model.Resolver) model.Resolver {
	if len(cache) > 0 {
		reso = &cacheResolver{
			cache:    cache,
			mu:       sync.Mutex{},
			readOnly: true,
			resolver: reso,
		}
	}
	return reso
}

// cacheResolver implements CachingResolver and StaticDNSCache.
type cacheResolver struct {
	// cache is the underlying DNS cache.
	cache map[string][]string

	// mu provides mutual exclusion.
	mu sync.Mutex

	// readOnly means that we won't cache the result of successful resolutions.
	readOnly bool

	// resolver is the underlying resolver.
	resolver model.Resolver
}

var _ model.Resolver = &cacheResolver{}

// LookupHost implements model.Resolver.LookupHost
func (r *cacheResolver) LookupHost(
	ctx context.Context, hostname string) ([]string, error) {
	if entry := r.get(hostname); entry != nil {
		return entry, nil
	}
	entry, err := r.resolver.LookupHost(ctx, hostname)
	if err != nil {
		return nil, err
	}
	if !r.readOnly {
		r.set(hostname, entry)
	}
	return entry, nil
}

// get gets the currently configured entry for domain, or nil
func (r *cacheResolver) get(domain string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cache[domain]
}

// set sets a valid inside the cache iff readOnly is false.
func (r *cacheResolver) set(domain string, addresses []string) {
	r.mu.Lock()
	if r.cache == nil {
		r.cache = make(map[string][]string)
	}
	r.cache[domain] = addresses
	r.mu.Unlock()
}

// Address implements model.Resolver.Address.
func (r *cacheResolver) Address() string {
	return r.resolver.Address()
}

// Network implements model.Resolver.Network.
func (r *cacheResolver) Network() string {
	return r.resolver.Network()
}

// CloseIdleConnections implements model.Resolver.CloseIdleConnections.
func (r *cacheResolver) CloseIdleConnections() {
	r.resolver.CloseIdleConnections()
}

// LookupHTTPS implements model.Resolver.LookupHTTPS.
func (r *cacheResolver) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return nil, ErrNoDNSTransport
}

// LookupNS implements model.Resolver.LookupNS.
func (r *cacheResolver) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	return nil, ErrNoDNSTransport
}
