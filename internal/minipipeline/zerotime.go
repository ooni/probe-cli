package minipipeline

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

// ZeroTimeDNSLookupResults MUTATES values in input to zero its T0 and T fields.
func ZeroTimeDNSLookupResults(values []*model.ArchivalDNSLookupResult) {
	for _, entry := range values {
		entry.T0 = 0
		entry.T = 0
	}
}

// ZeroTimeNetworkEvents is like [ZeroTimeDNSLookupResults] but for network events.
func ZeroTimeNetworkEvents(values []*model.ArchivalNetworkEvent) {
	for _, entry := range values {
		entry.T0 = 0
		entry.T = 0
	}
}

// ZeroTimeTCPConnectResults is like [ZeroTimeDNSLookupResults] but for TCP connect results.
func ZeroTimeTCPConnectResults(values []*model.ArchivalTCPConnectResult) {
	for _, entry := range values {
		entry.T0 = 0
		entry.T = 0
	}
}

// ZeroTimeTLSHandshakeResults is like [ZeroTimeDNSLookupResults] but for TLS handshake results.
func ZeroTimeTLSHandshakeResults(values []*model.ArchivalTLSOrQUICHandshakeResult) {
	for _, entry := range values {
		entry.T0 = 0
		entry.T = 0
	}
}
