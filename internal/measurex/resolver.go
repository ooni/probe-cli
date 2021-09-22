package measurex

//
// Resolver
//
// Wrappers for Resolver to store events into a WritableDB.
//

import (
	"context"
	"encoding/json"
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
	return &resolverDB{Resolver: r, db: db, begin: mx.Begin}
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

// LookupHostEvent contains the result of a host lookup.
type LookupHostEvent struct {
	Network  string
	Address  string
	Domain   string
	Started  float64
	Finished float64
	Error    error
	Oddity   Oddity
	Addrs    []string
}

// MarshalJSON marshals a LookupHostEvent to the archival
// format compatible with df-002-dnst.
func (ev *LookupHostEvent) MarshalJSON() ([]byte, error) {
	archival := NewArchivalLookupHostList(ev)
	return json.Marshal(archival)
}

func (r *resolverDB) LookupHost(ctx context.Context, domain string) ([]string, error) {
	started := time.Since(r.begin).Seconds()
	addrs, err := r.Resolver.LookupHost(ctx, domain)
	finished := time.Since(r.begin).Seconds()
	r.db.InsertIntoLookupHost(&LookupHostEvent{
		Network:  r.Resolver.Network(),
		Address:  r.Resolver.Address(),
		Domain:   domain,
		Started:  started,
		Finished: finished,
		Error:    err,
		Oddity:   r.computeOddityLookupHost(addrs, err),
		Addrs:    addrs,
	})
	return addrs, err
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

// LookupHTTPSSvcEvent contains the results of an HTTPSSvc lookup.
type LookupHTTPSSvcEvent struct {
	Network  string
	Address  string
	Domain   string
	Started  float64
	Finished float64
	Error    error
	Oddity   Oddity
	IPv4     []string
	IPv6     []string
	ALPN     []string
}

// MarshalJSON marshals a LookupHTTPSSvcEvent to the archival
// format that is similar to df-002-dnst.
func (ev *LookupHTTPSSvcEvent) MarshalJSON() ([]byte, error) {
	archival := NewArchivalLookupHTTPSSvcList(ev)
	return json.Marshal(archival)
}

func (r *resolverDB) LookupHTTPSSvcWithoutRetry(ctx context.Context, domain string) (HTTPSSvc, error) {
	started := time.Since(r.begin).Seconds()
	https, err := r.Resolver.LookupHTTPSSvcWithoutRetry(ctx, domain)
	finished := time.Since(r.begin).Seconds()
	ev := &LookupHTTPSSvcEvent{
		Network:  r.Resolver.Network(),
		Address:  r.Resolver.Address(),
		Domain:   domain,
		Started:  started,
		Finished: finished,
		Error:    err,
		Oddity:   Oddity(r.computeOddityHTTPSSvc(https, err)),
	}
	if err == nil {
		ev.IPv4 = https.IPv4Hint()
		ev.IPv6 = https.IPv6Hint()
		ev.ALPN = https.ALPN()
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
