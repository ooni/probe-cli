package tracex

//
// DNS lookup and round trip
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// SaverResolver is a resolver that saves events.
type SaverResolver struct {
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
	return &SaverResolver{
		Resolver: r,
		Saver:    s,
	}
}

// LookupHost implements Resolver.LookupHost
func (r *SaverResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	start := time.Now()
	r.Saver.Write(Event{
		Address:  r.Resolver.Address(),
		Hostname: hostname,
		Name:     "resolve_start",
		Proto:    r.Resolver.Network(),
		Time:     start,
	})
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	stop := time.Now()
	r.Saver.Write(Event{
		Addresses: addrs,
		Address:   r.Resolver.Address(),
		Duration:  stop.Sub(start),
		Err:       err,
		Hostname:  hostname,
		Name:      "resolve_done",
		Proto:     r.Resolver.Network(),
		Time:      stop,
	})
	return addrs, err
}

func (r *SaverResolver) Network() string {
	return r.Resolver.Network()
}

func (r *SaverResolver) Address() string {
	return r.Resolver.Address()
}

func (r *SaverResolver) CloseIdleConnections() {
	r.Resolver.CloseIdleConnections()
}

func (r *SaverResolver) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	// TODO(bassosimone): we should probably implement this method
	return r.Resolver.LookupHTTPS(ctx, domain)
}

func (r *SaverResolver) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	// TODO(bassosimone): we should probably implement this method
	return r.Resolver.LookupNS(ctx, domain)
}

// SaverDNSTransport is a DNS transport that saves events.
type SaverDNSTransport struct {
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
	return &SaverDNSTransport{
		DNSTransport: txp,
		Saver:        s,
	}
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp *SaverDNSTransport) RoundTrip(
	ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
	start := time.Now()
	txp.Saver.Write(Event{
		Address:  txp.DNSTransport.Address(),
		DNSQuery: dnsMaybeQueryBytes(query),
		Name:     "dns_round_trip_start",
		Proto:    txp.DNSTransport.Network(),
		Time:     start,
	})
	response, err := txp.DNSTransport.RoundTrip(ctx, query)
	stop := time.Now()
	txp.Saver.Write(Event{
		Address:  txp.DNSTransport.Address(),
		DNSQuery: dnsMaybeQueryBytes(query),
		DNSReply: dnsMaybeResponseBytes(response),
		Duration: stop.Sub(start),
		Err:      err,
		Name:     "dns_round_trip_done",
		Proto:    txp.DNSTransport.Network(),
		Time:     stop,
	})
	return response, err
}

func (txp *SaverDNSTransport) Network() string {
	return txp.DNSTransport.Network()
}

func (txp *SaverDNSTransport) Address() string {
	return txp.DNSTransport.Address()
}

func (txp *SaverDNSTransport) CloseIdleConnections() {
	txp.DNSTransport.CloseIdleConnections()
}

func (txp *SaverDNSTransport) RequiresPadding() bool {
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

var _ model.Resolver = &SaverResolver{}
var _ model.DNSTransport = &SaverDNSTransport{}
