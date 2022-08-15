package webconnectivity

import "sync"

// DNSCache wraps a model.Resolver to provide DNS caching.
//
// The zero value is invalid. Please, use NewDNSCache.
type DNSCache struct {
	// mu provides mutual exclusion.
	mu *sync.Mutex

	// values contains already resolved values.
	values map[string][]string
}

// Get gets values from the cache
func (c *DNSCache) Get(domain string) ([]string, bool) {
	c.mu.Lock()
	values, found := c.values[domain]
	c.mu.Unlock()
	return values, found
}

// Set inserts into the cache
func (c *DNSCache) Set(domain string, values []string) {
	c.mu.Lock()
	c.values[domain] = values
	c.mu.Unlock()
}

// NewDNSCache creates a new DNSCache instance.
func NewDNSCache() *DNSCache {
	return &DNSCache{
		mu:     &sync.Mutex{},
		values: map[string][]string{},
	}
}
