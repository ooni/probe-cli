package dslx

//
// Collecting observations
//

import (
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// Observations is the skeleton shared by most OONI measurements where
// we group observations by type using standard test keys.
type Observations struct {
	// NetworkEvents contains I/O events.
	NetworkEvents []*model.ArchivalNetworkEvent `json:"network_events"`

	// Queries contains the DNS queries results.
	Queries []*model.ArchivalDNSLookupResult `json:"queries"`

	// Requests contains HTTP request results.
	Requests []*model.ArchivalHTTPRequestResult `json:"requests"`

	// TCPConnect contains the TCP connect results.
	TCPConnect []*model.ArchivalTCPConnectResult `json:"tcp_connect"`

	// TLSHandshakes contains the TLS handshakes results.
	TLSHandshakes []*model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshakes"`

	// QUICHandshakes contains the QUIC handshakes results.
	QUICHandshakes []*model.ArchivalTLSOrQUICHandshakeResult `json:"quic_handshakes"`
}

// ExtractObservations extracts observations from a list of [Maybe].
func ExtractObservations[T any](rs ...*Maybe[T]) (out []*Observations) {
	for _, r := range rs {
		out = append(out, r.Observations...)
	}
	return
}

// maybeTraceToObservations returns the observations inside the
// trace taking into account the case where trace is nil.
func maybeTraceToObservations(trace *measurexlite.Trace) (out []*Observations) {
	if trace != nil {
		out = append(out, &Observations{
			NetworkEvents:  trace.NetworkEvents(),
			Queries:        trace.DNSLookupsFromRoundTrip(),
			Requests:       []*model.ArchivalHTTPRequestResult{}, // no extractor inside trace!
			TCPConnect:     trace.TCPConnects(),
			TLSHandshakes:  trace.TLSHandshakes(),
			QUICHandshakes: trace.QUICHandshakes(),
		})
	}
	return
}
