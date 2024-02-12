package minipipeline

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

// NormalizeDNSLookupResults MUTATES values in input to zero its T0 and T fields and applies
// other normalizations meant to reduce the size of diffs.
func NormalizeDNSLookupResults(values []*model.ArchivalDNSLookupResult) {
	for _, entry := range values {
		entry.T0 = 0
		entry.T = 0
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
	}
}
