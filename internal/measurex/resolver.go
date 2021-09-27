package measurex

//
// Resolver
//
// Wrappers for Resolver to store events into a WritableDB.
//

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
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
		logger, dnsx.NewSerialResolver(
			mx.WrapDNSXRoundTripper(db, dnsx.NewDNSOverUDP(
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

// DNSLookupAnswer is a DNS lookup answer.
type DNSLookupAnswer struct {
	// JSON names compatible with df-002-dnst's spec
	Type string `json:"answer_type"`
	IPv4 string `json:"ipv4,omitempty"`
	IPv6 string `json:"ivp6,omitempty"`

	// Names not part of the spec.
	ALPN string `json:"alpn,omitempty"`
}

// DNSLookupEvent contains the results of a DNS lookup.
type DNSLookupEvent struct {
	// fields inside df-002-dnst
	Answers   []DNSLookupAnswer `json:"answers"`
	Network   string            `json:"engine"`
	Failure   *string           `json:"failure"`
	Domain    string            `json:"hostname"`
	QueryType string            `json:"query_type"`
	Address   string            `json:"resolver_address"`
	Finished  float64           `json:"t"`

	// Names not part of the spec.
	Started float64 `json:"started"`
	Oddity  Oddity  `json:"oddity"`
}

// SupportsHTTP3 returns true if this query is for HTTPS and
// the answer contains an ALPN for "h3"
func (ev *DNSLookupEvent) SupportsHTTP3() bool {
	if ev.QueryType != "HTTPS" {
		return false
	}
	for _, ans := range ev.Answers {
		switch ans.Type {
		case "ALPN":
			if ans.ALPN == "h3" {
				return true
			}
		}
	}
	return false
}

// Addrs returns all the IPv4/IPv6 addresses
func (ev *DNSLookupEvent) Addrs() (out []string) {
	for _, ans := range ev.Answers {
		switch ans.Type {
		case "A":
			if net.ParseIP(ans.IPv4) != nil {
				out = append(out, ans.IPv4)
			}
		case "AAAA":
			if net.ParseIP(ans.IPv6) != nil {
				out = append(out, ans.IPv6)
			}
		}
	}
	return
}

func (r *resolverDB) LookupHost(ctx context.Context, domain string) ([]string, error) {
	started := time.Since(r.begin).Seconds()
	addrs, err := r.Resolver.LookupHost(ctx, domain)
	finished := time.Since(r.begin).Seconds()
	for _, qtype := range []string{"A", "AAAA"} {
		ev := &DNSLookupEvent{
			Answers:   r.computeAnswers(addrs, qtype),
			Network:   r.Resolver.Network(),
			Address:   r.Resolver.Address(),
			Failure:   NewArchivalFailure(err),
			Domain:    domain,
			QueryType: qtype,
			Finished:  finished,
			Started:   started,
			Oddity:    r.computeOddityLookupHost(addrs, err),
		}
		r.db.InsertIntoLookupHost(ev)
	}
	return addrs, err
}

func (r *resolverDB) computeAnswers(addrs []string, qtype string) (out []DNSLookupAnswer) {
	for _, addr := range addrs {
		if qtype == "A" && !strings.Contains(addr, ":") {
			out = append(out, DNSLookupAnswer{Type: qtype, IPv4: addr})
			continue
		}
		if qtype == "AAAA" && strings.Contains(addr, ":") {
			out = append(out, DNSLookupAnswer{Type: qtype, IPv6: addr})
			continue
		}
	}
	return
}

func (r *resolverDB) computeOddityLookupHost(addrs []string, err error) Oddity {
	if err != nil {
		switch err.Error() {
		case errorsx.FailureGenericTimeoutError:
			return OddityDNSLookupTimeout
		case errorsx.FailureDNSNXDOMAINError:
			return OddityDNSLookupNXDOMAIN
		case errorsx.FailureDNSRefusedError:
			return OddityDNSLookupRefused
		default:
			return OddityDNSLookupOther
		}
	}
	for _, addr := range addrs {
		if isBogon(addr) {
			return OddityDNSLookupBogon
		}
	}
	return ""
}

func (r *resolverDB) LookupHTTPS(ctx context.Context, domain string) (HTTPSSvc, error) {
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
		Failure:   NewArchivalFailure(err),
		Oddity:    Oddity(r.computeOddityHTTPSSvc(https, err)),
	}
	if err == nil {
		for _, addr := range https.IPv4Hint() {
			ev.Answers = append(ev.Answers, DNSLookupAnswer{
				Type: "A",
				IPv4: addr,
			})
		}
		for _, addr := range https.IPv6Hint() {
			ev.Answers = append(ev.Answers, DNSLookupAnswer{
				Type: "AAAA",
				IPv6: addr,
			})
		}
		for _, alpn := range https.ALPN() {
			ev.Answers = append(ev.Answers, DNSLookupAnswer{
				Type: "ALPN",
				ALPN: alpn,
			})
		}
	}
	r.db.InsertIntoLookupHTTPSSvc(ev)
	return https, err
}

func (r *resolverDB) computeOddityHTTPSSvc(https HTTPSSvc, err error) Oddity {
	if err != nil {
		return r.computeOddityLookupHost(nil, err)
	}
	var addrs []string
	addrs = append(addrs, https.IPv4Hint()...)
	addrs = append(addrs, https.IPv6Hint()...)
	return r.computeOddityLookupHost(addrs, nil)
}
