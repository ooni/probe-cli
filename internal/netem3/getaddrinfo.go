package netem3

//
// Getaddrinfo implementation(s)
//

import (
	"context"
	"errors"
	"sync"

	"github.com/miekg/dns"
)

// StaticGetaddrinfoEntry is an entry used by [StaticGetaddrinfo].
type StaticGetaddrinfoEntry struct {
	// Addresses contains the resolved addresses.
	Addresses []string

	// CNAME contains the CNAME.
	CNAME string
}

// StaticGetaddrinfo implements [GetaddrinfoBackend]
// using a static map to lookup addresses. The zero value
// is invalid; instantiate using [NewStaticGetaddrinfo].
type StaticGetaddrinfo struct {
	// m is the static map.
	m map[string]*StaticGetaddrinfoEntry

	// mu is provides mutual exclusion.
	mu sync.Mutex
}

// NewStaticGetaddrinfo creates a new [StaticGetaddrinfo] instance.
func NewStaticGetaddrinfo() *StaticGetaddrinfo {
	return &StaticGetaddrinfo{
		m:  map[string]*StaticGetaddrinfoEntry{},
		mu: sync.Mutex{},
	}
}

// AddStaticEntry adds a [StaticGetaddrinfoEntry] to [StaticGetaddrinfo].
func (sg *StaticGetaddrinfo) AddStaticEntry(domain string, entry *StaticGetaddrinfoEntry) {
	sg.mu.Lock()
	sg.m[dns.CanonicalName(domain)] = entry
	sg.mu.Unlock()
}

var _ UNetGetaddrinfo = &StaticGetaddrinfo{}

// ErrDNSNoSuchHost is returned when a DNS lookup fails.
var ErrDNSNoSuchHost = errors.New("netem: dns: no such host")

// ErrDNSServerMisbehaving is the error we return when we don't
// know otherwise how to characterize the DNS failure.
var ErrDNSServerMisbehaving = errors.New("netem: dns: server misbehaving")

// Lookup implements GetaddrinfoBackend
func (sg *StaticGetaddrinfo) Lookup(ctx context.Context, domain string) ([]string, string, error) {
	defer sg.mu.Unlock()
	sg.mu.Lock()
	entry := sg.m[dns.CanonicalName(domain)]
	if entry == nil {
		return nil, "", ErrDNSNoSuchHost
	}
	if len(entry.Addresses) <= 0 {
		return nil, "", ErrDNSServerMisbehaving
	}
	return entry.Addresses, entry.CNAME, nil
}
