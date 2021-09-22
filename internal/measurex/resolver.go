package measurex

//
// Resolver
//
// Wrappers for netxlite's resolvers that are able
// to store events into an EventDB.
//

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// HTTPSSvc is the result returned by HTTPSSvc queries.
type HTTPSSvc = dnsx.HTTPSSvc

// Resolver is the resolver type we use. This resolver will
// store resolve events into the DB.
type Resolver interface {
	netxlite.Resolver
}

// WrapResolver wraps a Resolver so that we save measurements into the DB.
func WrapResolver(measurementID int64,
	origin Origin, db EventDB, r netxlite.Resolver) Resolver {
	return &resolverx{Resolver: r, db: db, origin: origin, mid: measurementID}
}

// NewResolverSystem is a convenience factory for creating a
// system resolver that saves measurements into a DB.
func NewResolverSystem(measurementID int64,
	origin Origin, db EventDB, logger Logger) Resolver {
	return WrapResolver(
		measurementID, origin, db, netxlite.NewResolverStdlib(logger))
}

// NewResolverUDP is a convenience factory for creating a Resolver
// using UDP that saves measurements into the DB.
//
// Arguments:
//
// - measurementID is the measurement ID;
//
// - origin is OrigiProbe or OriginTH;
//
// - db is where to save events;
//
// - logger is the logger;
//
// - address is the resolver address (e.g., "1.1.1.1:53").
func NewResolverUDP(measurementID int64,
	origin Origin, db EventDB, logger Logger, address string) Resolver {
	return WrapResolver(measurementID, origin, db,
		netxlite.WrapResolver(logger, dnsx.NewSerialResolver(
			WrapDNSXRoundTripper(measurementID, origin, db, dnsx.NewDNSOverUDP(
				&netxliteDialerAdapter{
					NewDialerWithSystemResolver(
						measurementID, origin, db, logger),
				},
				address,
			)))),
	)
}

type resolverx struct {
	netxlite.Resolver
	db     EventDB
	mid    int64
	origin Origin
}

// LookupHostEvent contains the result of a host lookup.
type LookupHostEvent struct {
	Origin        Origin
	MeasurementID int64
	ConnID        int64 // connID (typically zero)
	Network       string
	Address       string
	Domain        string
	Started       time.Duration
	Finished      time.Duration
	Error         error
	Oddity        Oddity
	Addrs         []string
}

func (r *resolverx) LookupHost(ctx context.Context, domain string) ([]string, error) {
	started := r.db.ElapsedTime()
	addrs, err := r.Resolver.LookupHost(ctx, domain)
	finished := r.db.ElapsedTime()
	r.db.InsertIntoLookupHost(&LookupHostEvent{
		Origin:        r.origin,
		MeasurementID: r.mid,
		Network:       r.Resolver.Network(),
		Address:       r.Resolver.Address(),
		Domain:        domain,
		Started:       started,
		Finished:      finished,
		Error:         err,
		Oddity:        r.computeOddityLookupHost(addrs, err),
		Addrs:         addrs,
	})
	return addrs, err
}

func (r *resolverx) computeOddityLookupHost(addrs []string, err error) Oddity {
	if err == nil {
		for _, addr := range addrs {
			if isBogon(addr) {
				return OddityDNSLookupBogon
			}
		}
		return ""
	}
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

// LookupHTTPSSvcEvent is the event emitted when we perform
// an HTTPSSvc DNS query for a domain.
type LookupHTTPSSvcEvent struct {
	Origin        Origin
	MeasurementID int64
	ConnID        int64 // connID (typically zero)
	Network       string
	Address       string
	Domain        string
	Started       time.Duration
	Finished      time.Duration
	Error         error
	Oddity        Oddity
	IPv4          []string
	IPv6          []string
	ALPN          []string
}

func (r *resolverx) LookupHTTPSSvcWithoutRetry(ctx context.Context, domain string) (HTTPSSvc, error) {
	started := r.db.ElapsedTime()
	https, err := r.Resolver.LookupHTTPSSvcWithoutRetry(ctx, domain)
	finished := r.db.ElapsedTime()
	ev := &LookupHTTPSSvcEvent{
		Origin:        r.origin,
		MeasurementID: r.mid,
		Network:       r.Resolver.Network(),
		Address:       r.Resolver.Address(),
		Domain:        domain,
		Started:       started,
		Finished:      finished,
		Error:         err,
		Oddity:        Oddity(r.computeOddityHTTPSSvc(https, err)),
	}
	if err == nil {
		ev.IPv4 = https.IPv4Hint()
		ev.IPv6 = https.IPv6Hint()
		ev.ALPN = https.ALPN()
	}
	r.db.InsertIntoLookupHTTPSSvc(ev)
	return https, err
}

func (r *resolverx) computeOddityHTTPSSvc(https HTTPSSvc, err error) Oddity {
	if err != nil {
		return r.computeOddityLookupHost(nil, err)
	}
	var addrs []string
	addrs = append(addrs, https.IPv4Hint()...)
	addrs = append(addrs, https.IPv6Hint()...)
	return r.computeOddityLookupHost(addrs, nil)
}
