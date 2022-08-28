package tracex

//
// DNS lookup and round trip
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// ResolverSaver is a resolver that saves events.
type ResolverSaver struct {
	// Resolver is the underlying resolver.
	Resolver model.Resolver

	// Saver saves events.
	Saver *Saver
}

// WrapResolver wraps a model.Resolver with a SaverResolver that will save
// the DNS lookup results into this Saver.
//
// When this function is invoked on a nil Saver, it will directly return
// the original Resolver without any wrapping.
func (s *Saver) WrapResolver(r model.Resolver) model.Resolver {
	if s == nil {
		return r
	}
	return &ResolverSaver{
		Resolver: r,
		Saver:    s,
	}
}

// LookupHost implements Resolver.LookupHost
func (r *ResolverSaver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	start := time.Now()
	r.Saver.Write(&EventResolveStart{&EventValue{
		Address:  r.Resolver.Address(),
		Hostname: hostname,
		Proto:    r.Network(),
		Time:     start,
	}})
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	stop := time.Now()
	r.Saver.Write(&EventResolveDone{&EventValue{
		Addresses: addrs,
		Address:   r.Resolver.Address(),
		Duration:  stop.Sub(start),
		Err:       NewFailureStr(err),
		Hostname:  hostname,
		Proto:     r.Network(),
		Time:      stop,
	}})
	return addrs, err
}

// ResolverNetworkAdaptNames makes sure we map the [netxlite.StdlibResolverGolangNetResolver] and
// [netxlite.StdlibResolverGetaddrinfo] resolver names to [netxlite.StdlibResolverSystem]. You MUST
// call this function when your resolver splits the "stdlib" resolver results into two fake AAAA
// and A queries rather than faking a single ANY query.
//
// See https://github.com/ooni/spec/pull/257 for more information.
func ResolverNetworkAdaptNames(input string) string {
	switch input {
	case netxlite.StdlibResolverGetaddrinfo, netxlite.StdlibResolverGolangNetResolver:
		return netxlite.StdlibResolverSystem
	default:
		return input
	}
}

func (r *ResolverSaver) Network() string {
	return ResolverNetworkAdaptNames(r.Resolver.Network())
}

func (r *ResolverSaver) Address() string {
	return r.Resolver.Address()
}

func (r *ResolverSaver) CloseIdleConnections() {
	r.Resolver.CloseIdleConnections()
}

func (r *ResolverSaver) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	// TODO(bassosimone): we should probably implement this method
	return r.Resolver.LookupHTTPS(ctx, domain)
}

func (r *ResolverSaver) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	// TODO(bassosimone): we should probably implement this method
	return r.Resolver.LookupNS(ctx, domain)
}

// DNSTransportSaver is a DNS transport that saves events.
type DNSTransportSaver struct {
	// DNSTransport is the underlying DNS transport.
	DNSTransport model.DNSTransport

	// Saver saves events.
	Saver *Saver
}

// WrapDNSTransport wraps a model.DNSTransport with a SaverDNSTransport that
// will save the DNS round trip results into this Saver.
//
// When this function is invoked on a nil Saver, it will directly return
// the original DNSTransport without any wrapping.
func (s *Saver) WrapDNSTransport(txp model.DNSTransport) model.DNSTransport {
	if s == nil {
		return txp
	}
	return &DNSTransportSaver{
		DNSTransport: txp,
		Saver:        s,
	}
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp *DNSTransportSaver) RoundTrip(
	ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
	start := time.Now()
	txp.Saver.Write(&EventDNSRoundTripStart{&EventValue{
		Address:  txp.DNSTransport.Address(),
		DNSQuery: dnsMaybeQueryBytes(query),
		Proto:    txp.Network(),
		Time:     start,
	}})
	response, err := txp.DNSTransport.RoundTrip(ctx, query)
	stop := time.Now()
	txp.Saver.Write(&EventDNSRoundTripDone{&EventValue{
		Address:     txp.DNSTransport.Address(),
		DNSQuery:    dnsMaybeQueryBytes(query),
		DNSResponse: dnsMaybeResponseBytes(response),
		Duration:    stop.Sub(start),
		Err:         NewFailureStr(err),
		Proto:       txp.Network(),
		Time:        stop,
	}})
	return response, err
}

func (txp *DNSTransportSaver) Network() string {
	return ResolverNetworkAdaptNames(txp.DNSTransport.Network())
}

func (txp *DNSTransportSaver) Address() string {
	return txp.DNSTransport.Address()
}

func (txp *DNSTransportSaver) CloseIdleConnections() {
	txp.DNSTransport.CloseIdleConnections()
}

func (txp *DNSTransportSaver) RequiresPadding() bool {
	return txp.DNSTransport.RequiresPadding()
}

func dnsMaybeQueryBytes(query model.DNSQuery) []byte {
	data, _ := query.Bytes()
	return data
}

func dnsMaybeResponseBytes(response model.DNSResponse) []byte {
	if response == nil {
		return nil
	}
	return response.Bytes()
}

var _ model.Resolver = &ResolverSaver{}
var _ model.DNSTransport = &DNSTransportSaver{}
