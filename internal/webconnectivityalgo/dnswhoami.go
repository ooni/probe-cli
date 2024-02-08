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
)

// DNSWhoamiInfoEntry contains an entry for DNSWhoamiInfo.
type DNSWhoamiInfoEntry struct {
	// Address is the IP address
	Address string `json:"address"`
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
	// mu provides mutual exclusion
	mu *sync.Mutex

	// systemv4 contains systemv4 results
	systemv4 []DNSWhoamiInfoEntry

	// udpv4 contains udpv4 results
	udpv4 map[string][]DNSWhoamiInfoEntry
}

// NewDNSWhoamiService constructs a new [*DNSWhoamiService].
func NewDNSWhoamiService() *DNSWhoamiService {
	return &DNSWhoamiService{
		mu:       &sync.Mutex{},
		systemv4: []DNSWhoamiInfoEntry{},
		udpv4:    map[string][]DNSWhoamiInfoEntry{},
	}
}

// SystemV4 returns the results of querying using the system resolver and IPv4.
func (svc *DNSWhoamiService) SystemV4(ctx context.Context) ([]DNSWhoamiInfoEntry, bool) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if len(svc.systemv4) <= 0 {
		ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
		defer cancel()
		netx := &netxlite.Netx{}
		reso := netx.NewStdlibResolver(model.DiscardLogger)
		addrs, err := reso.LookupHost(ctx, "whoami.v4.powerdns.org")
		if err != nil || len(addrs) < 1 {
			return nil, false
		}
		svc.systemv4 = []DNSWhoamiInfoEntry{{
			Address: addrs[0],
		}}
	}
	return svc.systemv4, len(svc.systemv4) > 0
}

// UDPv4 returns the results of querying a given UDP resolver and IPv4.
func (svc *DNSWhoamiService) UDPv4(ctx context.Context, address string) ([]DNSWhoamiInfoEntry, bool) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if len(svc.udpv4[address]) <= 0 {
		ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
		defer cancel()
		netx := &netxlite.Netx{}
		dialer := netxlite.NewDialerWithStdlibResolver(model.DiscardLogger)
		reso := netx.NewParallelUDPResolver(model.DiscardLogger, dialer, address)
		// TODO(bassosimone): this should actually only send an A query. Sending an AAAA
		// query is _way_ unnecessary since we know that only A is going to work.
		addrs, err := reso.LookupHost(ctx, "whoami.v4.powerdns.org")
		if err != nil || len(addrs) < 1 {
			return nil, false
		}
		svc.udpv4[address] = []DNSWhoamiInfoEntry{{
			Address: addrs[0],
		}}
	}
	value := svc.udpv4[address]
	return value, len(value) > 0
}
