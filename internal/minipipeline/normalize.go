package minipipeline

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

// NormalizeDNSLookupResults MUTATES values in input to zero its T0 and T fields and applies
// other normalizations meant to reduce the size of diffs.
func NormalizeDNSLookupResults(values []*model.ArchivalDNSLookupResult) {
	for _, entry := range values {
		switch entry.Engine {
		case "udp":
			entry.ResolverAddress = "1.1.1.1:53"
		case "doh":
			entry.ResolverAddress = "https://dns.google/dns-query"
		}
		entry.T0 = 0
		entry.T = 0
		entry.RawResponse = nil
	}
}

// NormalizeNetworkEvents is like [NormalizeDNSLookupResults] but for network events.
func NormalizeNetworkEvents(values []*model.ArchivalNetworkEvent) {
	for _, entry := range values {
		entry.T0 = 0
		entry.T = 0
	}
}

// NormalizeTCPConnectResults is like [NormalizeDNSLookupResults] but for TCP connect results.
func NormalizeTCPConnectResults(values []*model.ArchivalTCPConnectResult) {
	for _, entry := range values {
		entry.T0 = 0
		entry.T = 0
	}
}

// NormalizeTLSHandshakeResults is like [NormalizeDNSLookupResults] but for TLS handshake results.
func NormalizeTLSHandshakeResults(values []*model.ArchivalTLSOrQUICHandshakeResult) {
	for _, entry := range values {
		entry.T0 = 0
		entry.T = 0
		entry.PeerCertificates = nil
	}
}

// NormalizeHTTPRequestResults is like [NormalizeDNSLookupResults] but for HTTP requests.
func NormalizeHTTPRequestResults(values []*model.ArchivalHTTPRequestResult) {
	for _, entry := range values {
		entry.T0 = 0
		entry.T = 0

		// Avoid storing large bodies because that wastes repository space.
		//
		// See https://github.com/ooni/probe/issues/2677.
		const maxStoreBody = 12_000
		if len(entry.Response.Body) > maxStoreBody {
			entry.Response.Body = model.ArchivalScrubbedMaybeBinaryString("")
			entry.Response.BodyIsTruncated = true
		}
	}
}
