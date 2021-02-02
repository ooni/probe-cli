package resolver

import (
	"context"
	"sync"
)

// CacheResolver is a resolver that caches successful replies.
type CacheResolver struct {
	ReadOnly bool
	Resolver
	mu    sync.Mutex
	cache map[string][]string
}

// LookupHost implements Resolver.LookupHost
func (r *CacheResolver) LookupHost(
	ctx context.Context, hostname string) ([]string, error) {
	if entry := r.Get(hostname); entry != nil {
		return entry, nil
	}
	entry, err := r.Resolver.LookupHost(ctx, hostname)
	if err != nil {
		return nil, err
	}
	if r.ReadOnly == false {
		r.Set(hostname, entry)
	}
	return entry, nil
}

// Get gets the currently configured entry for domain, or nil
func (r *CacheResolver) Get(domain string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cache[domain]
}

// Set allows to pre-populate the cache
func (r *CacheResolver) Set(domain string, addresses []string) {
	r.mu.Lock()
	if r.cache == nil {
		r.cache = make(map[string][]string)
	}
	r.cache[domain] = addresses
	r.mu.Unlock()
}
