package measurexlite

//
// DNS Lookup with tracing
//

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// wrapResolver resolver wraps the passed resolver to save data into the trace
func (tx *Trace) wrapResolver(resolver model.Resolver) model.Resolver {
	return &resolverTrace{
		r:  resolver,
		tx: tx,
	}
}

// resolverTrace is a trace-aware resolver
type resolverTrace struct {
	r  model.Resolver
	tx *Trace
}

var _ model.Resolver = &resolverTrace{}

// Address implements model.Resolver.Address
func (r *resolverTrace) Address() string {
	return r.r.Address()
}

// Network implements model.Resolver.Network
func (r *resolverTrace) Network() string {
	return r.r.Network()
}

// CloseIdleConnections implements model.Resolver.CloseIdleConnections
func (r *resolverTrace) CloseIdleConnections() {
	r.r.CloseIdleConnections()
}

// LookupHost implements model.Resolver.LookupHost
func (r *resolverTrace) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	return r.r.LookupHost(netxlite.ContextWithTrace(ctx, r.tx), hostname)
}

// LookupHTTPS implements model.Resolver.LookupHTTPS
func (r *resolverTrace) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return r.r.LookupHTTPS(netxlite.ContextWithTrace(ctx, r.tx), domain)
}

// LookupNS implements model.Resolver.LookupNS
func (r *resolverTrace) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	return r.r.LookupNS(netxlite.ContextWithTrace(ctx, r.tx), domain)
}

// NewStdlibResolver returns a trace-ware system resolver
func (tx *Trace) NewStdlibResolver(logger model.Logger) model.Resolver {
	return tx.wrapResolver(tx.newStdlibResolver(logger))
}

// NewParallelUDPResolver returns a trace-ware parallel UDP resolver
func (tx *Trace) NewParallelUDPResolver(logger model.Logger, dialer model.Dialer, address string) model.Resolver {
	return tx.wrapResolver(tx.newParallelUDPResolver(logger, dialer, address))
}

// NewParallelDNSOverHTTPSResolver returns a trace-aware parallel DoH resolver
func (tx *Trace) NewParallelDNSOverHTTPSResolver(logger model.Logger, URL string) model.Resolver {
	return tx.wrapResolver(tx.newParallelDNSOverHTTPSResolver(logger, URL))
}

// OnDNSRoundTripForLookupHost implements model.Trace.OnDNSRoundTripForLookupHost
func (tx *Trace) OnDNSRoundTripForLookupHost(started time.Time, reso model.Resolver, query model.DNSQuery,
	response model.DNSResponse, addrs []string, err error, finished time.Time) {
	t := finished.Sub(tx.ZeroTime)
	select {
	case tx.dnsLookup <- NewArchivalDNSLookupResultFromRoundTrip(
		tx.Index,
		started.Sub(tx.ZeroTime),
		reso,
		query,
		response,
		addrs,
		err,
		t,
	):
	default:
	}
}

// NewArchivalDNSLookupResultFromRoundTrip generates a model.ArchivalDNSLookupResultFromRoundTrip
// from the available information right after the DNS RoundTrip
func NewArchivalDNSLookupResultFromRoundTrip(index int64, started time.Duration, reso model.Resolver, query model.DNSQuery,
	response model.DNSResponse, addrs []string, err error, finished time.Duration) *model.ArchivalDNSLookupResult {
	return &model.ArchivalDNSLookupResult{
		Answers:          archivalAnswersFromAddrs(addrs),
		Engine:           reso.Network(),
		Failure:          tracex.NewFailure(err),
		Hostname:         query.Domain(),
		QueryType:        dns.TypeToString[query.Type()],
		ResolverHostname: nil,
		ResolverAddress:  reso.Address(),
		T:                finished.Seconds(),
	}
}

// archivalAnswersFromAddrs generates model.ArchivalDNSAnswer from an array of addresses
func archivalAnswersFromAddrs(addrs []string) (out []model.ArchivalDNSAnswer) {
	for _, addr := range addrs {
		ipv6, err := netxlite.IsIPv6(addr)
		if err != nil {
			log.Printf("BUG: NewArchivalDNSLookupResult: invalid IP address: %s", addr)
			continue
		}
		asn, org, _ := geolocate.LookupASN(addr)
		switch ipv6 {
		case false:
			out = append(out, model.ArchivalDNSAnswer{
				ASN:        int64(asn),
				ASOrgName:  org,
				AnswerType: "A",
				Hostname:   "",
				IPv4:       addr,
				TTL:        nil,
			})
		case true:
			out = append(out, model.ArchivalDNSAnswer{
				ASN:        int64(asn),
				ASOrgName:  org,
				AnswerType: "AAAA",
				Hostname:   "",
				IPv6:       addr,
				TTL:        nil,
			})
		}
	}
	return
}

// DNSLookupsFromRoundTrip drains the network events buffered inside the DNSLookup channel
func (tx *Trace) DNSLookupsFromRoundTrip() (out []*model.ArchivalDNSLookupResult) {
	for {
		select {
		case ev := <-tx.dnsLookup:
			out = append(out, ev)
		default:
			return
		}
	}
}

// FirstDNSLookup drains the network events buffered inside the DNSLookup channel
// and returns the first DNSLookup.
func (tx *Trace) FirstDNSLookup() *model.ArchivalDNSLookupResult {
	ev := tx.DNSLookupsFromRoundTrip()
	if len(ev) < 1 {
		return nil
	}
	return ev[0]
}
