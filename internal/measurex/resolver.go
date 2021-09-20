package measurex

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// HTTPSSvc is the result returned by HTTPSSvc queries.
type HTTPSSvc = dnsx.HTTPSSvc

// Resolver is the resolver type we use.
type Resolver interface {
	netxlite.Resolver
}

// WrapResolver wraps a netxlite.Resolver to add measurex capabilities.
func WrapResolver(origin Origin, db DB, r netxlite.Resolver) Resolver {
	return &resolverx{Resolver: r, db: db, origin: origin}
}

type resolverx struct {
	netxlite.Resolver
	db     DB
	origin Origin
}

// LookupHostEvent contains the result of a host lookup.
type LookupHostEvent struct {
	Origin        Origin
	MeasurementID int64
	Network       string
	Address       string
	Domain        string
	Started       time.Time
	Finished      time.Time
	Error         error
	Oddity        Oddity
	Addrs         []string
}

func (r *resolverx) LookupHost(ctx context.Context, domain string) ([]string, error) {
	started := time.Now()
	addrs, err := r.Resolver.LookupHost(ctx, domain)
	finished := time.Now()
	r.db.InsertIntoLookupHost(&LookupHostEvent{
		Origin:        r.origin,
		MeasurementID: r.db.MeasurementID(),
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
			if IsBogon(addr) {
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
	Domain        string
	Started       time.Time
	Finished      time.Time
	Error         error
	Oddity        Oddity
	IPv4          []string
	IPv6          []string
	ALPN          []string
}

func (r *resolverx) LookupHTTPSSvcWithoutRetry(ctx context.Context, domain string) (HTTPSSvc, error) {
	started := time.Now()
	https, err := r.Resolver.LookupHTTPSSvcWithoutRetry(ctx, domain)
	finished := time.Now()
	ev := &LookupHTTPSSvcEvent{
		Origin:        r.origin,
		MeasurementID: r.db.MeasurementID(),
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
