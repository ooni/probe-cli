package webconnectivityalgo

//
// DNS whoami lookups
//
// The purpose of this lookups is figuring out who's answering our DNS
// queries so we know whether there's interception ongoing.
//

import (
	"context"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// DNSWhoamiInfoEntry contains an entry for DNSWhoamiInfo.
type DNSWhoamiInfoEntry struct {
	// Address is the IP address used by the resolver.
	Address string `json:"address"`
}

// dnsWhoamiInfoTimedEntry keeps an address and the time we create the entry together.
type dnsWhoamiInfoTimedEntry struct {
	Addr string
	T    time.Time
}

// TODO(bassosimone): this code needs refining before we can merge it inside
// master. For one, we already have systemv4 info. Additionally, it would
// be neat to avoid additional AAAA queries. Furthermore, we should also see
// to implement support for IPv6 only clients as well.

// TODO(bassosimone): consider factoring this code and keeping state
// on disk rather than on memory.

// DNSWhoamiService is a service that performs DNS whoami lookups.
//
// The zero value of this struct is invalid. Please, construct using
// the [NewDNSWhoamiService] factory function.
type DNSWhoamiService struct {
	// entries contains the entries.
	entries map[string]*dnsWhoamiInfoTimedEntry

	// logger is the logger.
	logger model.Logger

	// mu provides mutual exclusion.
	mu *sync.Mutex

	// netx is the underlying network we're using.
	netx *netxlite.Netx

	// timeNow allows to get the current time.
	timeNow func() time.Time

	// whoamiDomain is the whoamiDomain to query for.
	whoamiDomain string
}

// NewDNSWhoamiService constructs a new [*DNSWhoamiService].
func NewDNSWhoamiService(logger model.Logger) *DNSWhoamiService {
	return &DNSWhoamiService{
		entries:      map[string]*dnsWhoamiInfoTimedEntry{},
		logger:       logger,
		mu:           &sync.Mutex{},
		netx:         &netxlite.Netx{Underlying: nil},
		timeNow:      time.Now,
		whoamiDomain: "whoami.v4.powerdns.org",
	}
}

// SystemV4 returns the results of querying using the system resolver and IPv4.
func (svc *DNSWhoamiService) SystemV4(ctx context.Context) ([]DNSWhoamiInfoEntry, bool) {
	spec := &dnsWhoamiResolverSpec{
		name: "system:///",
		factory: func(logger model.Logger, netx *netxlite.Netx) model.Resolver {
			return svc.netx.NewStdlibResolver(svc.logger)
		},
	}
	v := svc.lookup(ctx, spec)
	return v, len(v) > 0
}

// UDPv4 returns the results of querying a given UDP resolver and IPv4.
func (svc *DNSWhoamiService) UDPv4(ctx context.Context, address string) ([]DNSWhoamiInfoEntry, bool) {
	spec := &dnsWhoamiResolverSpec{
		name: address,
		factory: func(logger model.Logger, netx *netxlite.Netx) model.Resolver {
			dialer := svc.netx.NewDialerWithResolver(svc.logger, svc.netx.NewStdlibResolver(svc.logger))
			return svc.netx.NewParallelUDPResolver(svc.logger, dialer, address)
		},
	}
	v := svc.lookup(ctx, spec)
	return v, len(v) > 0
}

type dnsWhoamiResolverSpec struct {
	name    string
	factory func(logger model.Logger, netx *netxlite.Netx) model.Resolver
}

func (svc *DNSWhoamiService) lookup(ctx context.Context, spec *dnsWhoamiResolverSpec) []DNSWhoamiInfoEntry {
	// get the current time
	now := svc.timeNow()

	// possibly use cache
	mentry := svc.lockAndGet(now, spec.name)
	if !mentry.IsNone() {
		return []DNSWhoamiInfoEntry{mentry.Unwrap()}
	}

	// perform lookup
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	reso := spec.factory(svc.logger, svc.netx)
	addrs, err := reso.LookupHost(ctx, svc.whoamiDomain)
	if err != nil || len(addrs) < 1 {
		return nil
	}

	// update cache
	svc.lockAndUpdate(now, spec.name, addrs[0])

	// return to the caller
	return []DNSWhoamiInfoEntry{{Address: addrs[0]}}
}

func (svc *DNSWhoamiService) lockAndGet(now time.Time, serverAddr string) optional.Value[DNSWhoamiInfoEntry] {
	// ensure there's mutual exclusion
	defer svc.mu.Unlock()
	svc.mu.Lock()

	// see if there's an entry
	entry, found := svc.entries[serverAddr]
	if !found {
		return optional.None[DNSWhoamiInfoEntry]()
	}

	// make sure the entry has not expired
	const validity = 45 * time.Second
	if now.Sub(entry.T) > validity {
		return optional.None[DNSWhoamiInfoEntry]()
	}

	// return a copy of the value
	return optional.Some(DNSWhoamiInfoEntry{
		Address: entry.Addr,
	})
}

func (svc *DNSWhoamiService) lockAndUpdate(now time.Time, serverAddr, whoamiAddr string) {
	// ensure there's mutual exclusion
	defer svc.mu.Unlock()
	svc.mu.Lock()

	// insert into the table
	svc.entries[serverAddr] = &dnsWhoamiInfoTimedEntry{
		Addr: whoamiAddr,
		T:    now,
	}
}

func (svc *DNSWhoamiService) cloneEntries() map[string]*dnsWhoamiInfoTimedEntry {
	defer svc.mu.Unlock()
	svc.mu.Lock()
	output := make(map[string]*dnsWhoamiInfoTimedEntry)
	for key, value := range svc.entries {
		output[key] = &dnsWhoamiInfoTimedEntry{
			Addr: value.Addr,
			T:    value.T,
		}
	}
	return output
}
