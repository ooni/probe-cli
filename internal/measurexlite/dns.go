package measurexlite

//
// DNS Lookup with tracing
//

import (
	"context"
	"errors"
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

// DNSNetworkAddresser is the type of something we just used to perform a DNS
// round trip (e.g., model.DNSTransport, model.Resolver) that allows us to get
// the network and the address of the underlying resolver/transport.
type DNSNetworkAddresser interface {
	// Address is like model.DNSTransport.Address
	Address() string

	// Network is like model.DNSTransport.Network
	Network() string
}

// NewArchivalDNSLookupResultFromRoundTrip generates a model.ArchivalDNSLookupResultFromRoundTrip
// from the available information right after the DNS RoundTrip
func NewArchivalDNSLookupResultFromRoundTrip(index int64, started time.Duration, reso DNSNetworkAddresser, query model.DNSQuery,
	response model.DNSResponse, addrs []string, err error, finished time.Duration) *model.ArchivalDNSLookupResult {
	return &model.ArchivalDNSLookupResult{
		Answers:          newArchivalDNSAnswers(addrs, response),
		Engine:           reso.Network(),
		Failure:          tracex.NewFailure(err),
		Hostname:         query.Domain(),
		QueryType:        dns.TypeToString[query.Type()],
		ResolverHostname: nil,
		ResolverAddress:  reso.Address(),
		T:                finished.Seconds(),
	}
}

// newArchivalDNSAnswers generates []model.ArchivalDNSAnswer from [addrs] and [resp].
func newArchivalDNSAnswers(addrs []string, resp model.DNSResponse) (out []model.ArchivalDNSAnswer) {
	// Design note: in principle we might want to extract everything from the
	// response but, when we're called by netxlite, netxlite has already extracted
	// the addresses to return them to the caller, so I think it's fine to keep
	// this extraction code as such rather than suppressing passing the addrs from
	// netxlite. Also, a wrong IP address is a bug because netxlite should not
	// return invalid IP addresses from its resolvers, so we want to know about that.

	// Include IP addresses extracted by netxlite
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
				IPv6:       "",
				TTL:        nil,
			})
		case true:
			out = append(out, model.ArchivalDNSAnswer{
				ASN:        int64(asn),
				ASOrgName:  org,
				AnswerType: "AAAA",
				Hostname:   "",
				IPv4:       "",
				IPv6:       addr,
				TTL:        nil,
			})
		}
	}

	// Include additional answer types when a response is available
	if resp != nil {

		// Include CNAME if available
		if cname, err := resp.DecodeCNAME(); err == nil && cname != "" {
			out = append(out, model.ArchivalDNSAnswer{
				ASN:        0,
				ASOrgName:  "",
				AnswerType: "CNAME",
				Hostname:   cname,
				IPv4:       "",
				IPv6:       "",
				TTL:        nil,
			})
		}

		// TODO(bassosimone): what other fields generally present inside A/AAAA replies
		// would it be useful to extract here? Perhaps, the SoA field?
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

// FirstDNSLookupOrNil drains the network events buffered inside the DNSLookup channel
// and returns the first DNSLookup, if any. Otherwise, it returns nil.
func (tx *Trace) FirstDNSLookup() *model.ArchivalDNSLookupResult {
	ev := tx.DNSLookupsFromRoundTrip()
	if len(ev) < 1 {
		return nil
	}
	return ev[0]
}

// ErrDelayedDNSResponseBufferFull indicates that the delayedDNSResponse buffer is full.
var ErrDelayedDNSResponseBufferFull = errors.New("buffer full")

// OnDelayedDNSResponse implements model.Trace.OnDelayedDNSResponse
func (tx *Trace) OnDelayedDNSResponse(started time.Time, txp model.DNSTransport, query model.DNSQuery,
	response model.DNSResponse, addrs []string, err error, finished time.Time) error {
	t := finished.Sub(tx.ZeroTime)
	select {
	case tx.delayedDNSResponse <- NewArchivalDNSLookupResultFromRoundTrip(
		tx.Index,
		started.Sub(tx.ZeroTime),
		txp,
		query,
		response,
		addrs,
		err,
		t,
	):
		return nil
	default:
		return ErrDelayedDNSResponseBufferFull
	}
}

// DelayedDNSResponseWithTimeout drains the network events buffered inside
// the delayedDNSResponse channel. We construct a child context based on [ctx]
// and the given [timeout] and we stop reading when original [ctx] has been
// cancelled or the given [timeout] expires, whatever happens first. Once the
// timeout expired, we drain the chan as much as possible before returning.
func (tx *Trace) DelayedDNSResponseWithTimeout(ctx context.Context,
	timeout time.Duration) (out []*model.ArchivalDNSLookupResult) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			for { // once the context is done enter in channel draining mode
				select {
				case ev := <-tx.delayedDNSResponse:
					out = append(out, ev)
				default:
					return
				}
			}
		case ev := <-tx.delayedDNSResponse:
			out = append(out, ev)
		}
	}
}
