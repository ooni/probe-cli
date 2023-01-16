package webconnectivitylte

import "sync"

// DNSEntry is an entry in the DNS cache.
type DNSEntry struct {
	// Addr is the cached address
	Addr string

	// Flags contains flags
	Flags int64
}

const (
	// DNSAddrFlagSystemResolver means we discovered this addr using the system resolver.
	DNSAddrFlagSystemResolver = 1 << iota

	// DNSAddrFlagUDP means we discovered this addr using the UDP resolver.
	DNSAddrFlagUDP

	// DNSAddrFlagHTTPS means we discovered this addr using the DNS-over-HTTPS resolver.
	DNSAddrFlagHTTPS
)

// DNSCache wraps a model.Resolver to provide DNS caching.
//
// The zero value is invalid; please, use NewDNSCache to construct.
type DNSCache struct {
	// mu provides mutual exclusion.
	mu *sync.Mutex

	// values contains already resolved values.
	values map[string][]DNSEntry
}

// Get gets values from the cache
func (c *DNSCache) Get(domain string) ([]DNSEntry, bool) {
	c.mu.Lock()
	values, found := c.values[domain]
	c.mu.Unlock()
	return values, found
}

// Set inserts into the cache
func (c *DNSCache) Set(domain string, values []DNSEntry) {
	c.mu.Lock()
	c.values[domain] = values
	c.mu.Unlock()
}

// NewDNSCache creates a new DNSCache instance.
func NewDNSCache() *DNSCache {
	return &DNSCache{
		mu:     &sync.Mutex{},
		values: map[string][]DNSEntry{},
	}
}
