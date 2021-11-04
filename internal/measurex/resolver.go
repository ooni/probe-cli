package measurex

//
// Resolver
//
// Wrappers for Resolver to store events into a WritableDB.
//

import (
	"context"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
)

// HTTPSSvc is the result returned by HTTPSSvc queries.
type HTTPSSvc = dnsx.HTTPSSvc

// Resolver is the resolver type we use. This resolver will
// store resolve events into the DB.
type Resolver = netxlite.Resolver

// WrapResolver creates a new Resolver that saves events into the WritableDB.
func (mx *Measurer) WrapResolver(db WritableDB, r netxlite.Resolver) Resolver {
	return WrapResolver(mx.Begin, db, r)
}

// WrapResolver wraps a resolver.
func WrapResolver(begin time.Time, db WritableDB, r netxlite.Resolver) Resolver {
	return &resolverDB{Resolver: r, db: db, begin: begin}
}

// NewResolverSystem creates a system resolver and then wraps
// it using the WrapResolver function/
func (mx *Measurer) NewResolverSystem(db WritableDB, logger Logger) Resolver {
	return mx.WrapResolver(db, netxlite.NewResolverStdlib(logger))
}

// NewResolverUDP is a convenience factory for creating a Resolver
// using UDP that saves measurements into the DB.
//
// Arguments:
//
// - db is where to save events;
//
// - logger is the logger;
//
// - address is the resolver address (e.g., "1.1.1.1:53").
func (mx *Measurer) NewResolverUDP(db WritableDB, logger Logger, address string) Resolver {
	return mx.WrapResolver(db, netxlite.WrapResolver(
		logger, netxlite.NewSerialResolver(
			mx.WrapDNSXRoundTripper(db, netxlite.NewDNSOverUDP(
				mx.NewDialerWithSystemResolver(db, logger),
				address,
			)))),
	)
}

type resolverDB struct {
	netxlite.Resolver
	begin time.Time
	db    WritableDB
}

// DNSLookupEvent contains the results of a DNS lookup.
type DNSLookupEvent struct {
	Network   string
	Failure   *string
	Domain    string
	QueryType string
	Address   string
	Finished  float64
	Started   float64
	Oddity    Oddity
	A         []string
	AAAA      []string
	ALPN      []string
}

// SupportsHTTP3 returns true if this query is for HTTPS and
// the answer contains an ALPN for "h3"
func (ev *DNSLookupEvent) SupportsHTTP3() bool {
	if ev.QueryType != "HTTPS" {
		return false
	}
	for _, alpn := range ev.ALPN {
		if alpn == "h3" {
			return true
		}
	}
	return false
}

// Addrs returns all the IPv4/IPv6 addresses
func (ev *DNSLookupEvent) Addrs() (out []string) {
	out = append(out, ev.A...)
	out = append(out, ev.AAAA...)
	return
}

func (r *resolverDB) LookupHost(ctx context.Context, domain string) ([]string, error) {
	started := time.Since(r.begin).Seconds()
	addrs, err := r.Resolver.LookupHost(ctx, domain)
	finished := time.Since(r.begin).Seconds()
	r.saveLookupResults(domain, started, finished, err, addrs, "A")
	r.saveLookupResults(domain, started, finished, err, addrs, "AAAA")
	return addrs, err
}

func (r *resolverDB) saveLookupResults(domain string, started, finished float64,
	err error, addrs []string, qtype string) {
	ev := &DNSLookupEvent{
		Network:   r.Resolver.Network(),
		Address:   r.Resolver.Address(),
		Failure:   NewFailure(err),
		Domain:    domain,
		QueryType: qtype,
		Finished:  finished,
		Started:   started,
	}
	for _, addr := range addrs {
		if qtype == "A" && !strings.Contains(addr, ":") {
			ev.A = append(ev.A, addr)
			continue
		}
		if qtype == "AAAA" && strings.Contains(addr, ":") {
			ev.AAAA = append(ev.AAAA, addr)
			continue
		}
	}
	switch qtype {
	case "A":
		ev.Oddity = r.computeOddityLookupHost(ev.A, err)
	case "AAAA":
		ev.Oddity = r.computeOddityLookupHost(ev.AAAA, err)
	}
	r.db.InsertIntoLookupHost(ev)
}

func (r *resolverDB) computeOddityLookupHost(addrs []string, err error) Oddity {
	if err != nil {
		switch err.Error() {
		case netxlite.FailureGenericTimeoutError:
			return OddityDNSLookupTimeout
		case netxlite.FailureDNSNXDOMAINError:
			return OddityDNSLookupNXDOMAIN
		case netxlite.FailureDNSRefusedError:
			return OddityDNSLookupRefused
		default:
			return OddityDNSLookupOther
		}
	}
	for _, addr := range addrs {
		if netxlite.IsBogon(addr) {
			return OddityDNSLookupBogon
		}
	}
	return ""
}

func (r *resolverDB) LookupHTTPS(ctx context.Context, domain string) (*HTTPSSvc, error) {
	started := time.Since(r.begin).Seconds()
	https, err := r.Resolver.LookupHTTPS(ctx, domain)
	finished := time.Since(r.begin).Seconds()
	ev := &DNSLookupEvent{
		Network:   r.Resolver.Network(),
		Address:   r.Resolver.Address(),
		Domain:    domain,
		QueryType: "HTTPS",
		Started:   started,
		Finished:  finished,
		Failure:   NewFailure(err),
		Oddity:    Oddity(r.computeOddityHTTPSSvc(https, err)),
	}
	if err == nil {
		ev.A = append(ev.A, https.IPv4...)
		ev.AAAA = append(ev.AAAA, https.IPv6...)
		ev.ALPN = append(ev.ALPN, https.ALPN...)
	}
	r.db.InsertIntoLookupHTTPSSvc(ev)
	return https, err
}

func (r *resolverDB) computeOddityHTTPSSvc(https *HTTPSSvc, err error) Oddity {
	if err != nil {
		return r.computeOddityLookupHost(nil, err)
	}
	var addrs []string
	addrs = append(addrs, https.IPv4...)
	addrs = append(addrs, https.IPv6...)
	return r.computeOddityLookupHost(addrs, nil)
}
