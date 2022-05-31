package tracex

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// SaverResolver is a resolver that saves events
type SaverResolver struct {
	model.Resolver
	Saver *Saver
}

// LookupHost implements Resolver.LookupHost
func (r SaverResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
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

// SaverDNSTransport is a DNS transport that saves events
type SaverDNSTransport struct {
	model.DNSTransport
	Saver *Saver
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp SaverDNSTransport) RoundTrip(
	ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
	start := time.Now()
	txp.Saver.Write(Event{
		Address:  txp.Address(),
		DNSQuery: txp.maybeQueryBytes(query),
		Name:     "dns_round_trip_start",
		Proto:    txp.Network(),
		Time:     start,
	})
	response, err := txp.DNSTransport.RoundTrip(ctx, query)
	stop := time.Now()
	txp.Saver.Write(Event{
		Address:  txp.Address(),
		DNSQuery: txp.maybeQueryBytes(query),
		DNSReply: txp.maybeResponseBytes(response),
		Duration: stop.Sub(start),
		Err:      err,
		Name:     "dns_round_trip_done",
		Proto:    txp.Network(),
		Time:     stop,
	})
	return response, err
}

func (txp SaverDNSTransport) maybeQueryBytes(query model.DNSQuery) []byte {
	data, _ := query.Bytes()
	return data
}

func (txp SaverDNSTransport) maybeResponseBytes(response model.DNSResponse) []byte {
	if response == nil {
		return nil
	}
	return response.Bytes()
}

var _ model.Resolver = SaverResolver{}
var _ model.DNSTransport = SaverDNSTransport{}
