package archival

//
// Saves DNS lookup events
//

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSLookupEvent contains the results of a DNS lookup.
type DNSLookupEvent struct {
	ALPNs           []string
	Addresses       []string
	Domain          string
	Failure         error
	Finished        time.Time
	LookupType      string
	ResolverAddress string
	ResolverNetwork string
	Started         time.Time
}

// LookupHost performs a host lookup with the given resolver
// and saves the results into the saver.
func (s *Saver) LookupHost(ctx context.Context, reso model.Resolver, domain string) ([]string, error) {
	started := time.Now()
	addrs, err := reso.LookupHost(ctx, domain)
	s.appendLookupHostEvent(&DNSLookupEvent{
		ALPNs:           nil,
		Addresses:       addrs,
		Domain:          domain,
		Failure:         err,
		Finished:        time.Now(),
		LookupType:      "getaddrinfo",
		ResolverAddress: reso.Address(),
		ResolverNetwork: reso.Network(),
		Started:         started,
	})
	return addrs, err
}

func (s *Saver) appendLookupHostEvent(ev *DNSLookupEvent) {
	s.mu.Lock()
	s.trace.DNSLookupHost = append(s.trace.DNSLookupHost, ev)
	s.mu.Unlock()
}

// LookupHTTPS performs an HTTPSSvc-record lookup using the given
// resolver and saves the results into the saver.
func (s *Saver) LookupHTTPS(ctx context.Context, reso model.Resolver, domain string) (*model.HTTPSSvc, error) {
	started := time.Now()
	https, err := reso.LookupHTTPS(ctx, domain)
	s.appendLookupHTTPSEvent(&DNSLookupEvent{
		ALPNs:           s.safeALPNs(https),
		Addresses:       s.safeAddresses(https),
		Domain:          domain,
		Failure:         err,
		Finished:        time.Now(),
		LookupType:      "https",
		ResolverAddress: reso.Address(),
		ResolverNetwork: reso.Network(),
		Started:         started,
	})
	return https, err
}

func (s *Saver) appendLookupHTTPSEvent(ev *DNSLookupEvent) {
	s.mu.Lock()
	s.trace.DNSLookupHTTPS = append(s.trace.DNSLookupHTTPS, ev)
	s.mu.Unlock()
}

func (s *Saver) safeALPNs(https *model.HTTPSSvc) (out []string) {
	if https != nil {
		out = https.ALPN
	}
	return
}

func (s *Saver) safeAddresses(https *model.HTTPSSvc) (out []string) {
	if https != nil {
		out = append(out, https.IPv4...)
		out = append(out, https.IPv6...)
	}
	return
}

// DNSRoundTripEvent contains the result of a DNS round trip.
type DNSRoundTripEvent struct {
	Address  string
	Failure  error
	Finished time.Time
	Network  string
	Query    []byte
	Reply    []byte
	Started  time.Time
}

// DNSRoundTrip implements ArchivalSaver.DNSRoundTrip.
func (s *Saver) DNSRoundTrip(ctx context.Context, txp model.DNSTransport, query []byte) ([]byte, error) {
	started := time.Now()
	reply, err := txp.RoundTrip(ctx, query)
	s.appendDNSRoundTripEvent(&DNSRoundTripEvent{
		Address:  txp.Address(),
		Failure:  err,
		Finished: time.Now(),
		Network:  txp.Network(),
		Query:    query,
		Reply:    reply,
		Started:  started,
	})
	return reply, err
}

func (s *Saver) appendDNSRoundTripEvent(ev *DNSRoundTripEvent) {
	s.mu.Lock()
	s.trace.DNSRoundTrip = append(s.trace.DNSRoundTrip, ev)
	s.mu.Unlock()
}

// WrapResolver takes in input a resolver and returns in output
// a new one that uses this saver for saving events.
func (s *Saver) WrapResolver(r model.Resolver) model.Resolver {
	return &resolverSaver{Resolver: r, s: s}
}

type resolverSaver struct {
	model.Resolver
	s *Saver
}

func (r *resolverSaver) LookupHost(ctx context.Context, domain string) ([]string, error) {
	return r.s.LookupHost(ctx, r.Resolver, domain)
}

func (r *resolverSaver) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return r.s.LookupHTTPS(ctx, r.Resolver, domain)
}

// WrapDNSTransport takes in input a DNS transport and returns
// in output a new one using this saver for saving events.
func (s *Saver) WrapDNSTransport(txp model.DNSTransport) model.DNSTransport {
	return &dnsTransportSaver{DNSTransport: txp, s: s}
}

type dnsTransportSaver struct {
	model.DNSTransport
	s *Saver
}

func (txp *dnsTransportSaver) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	return txp.s.DNSRoundTrip(ctx, txp.DNSTransport, query)
}
