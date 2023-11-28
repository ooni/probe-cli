package minipipeline

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// WebMeasurement is the canonical web measurement structure assumed by minipipeline.
type WebMeasurement struct {
	// Input contains the input we measured (a URL).
	Input string `json:"input"`

	// TestKeys contains the test-specific measurements.
	TestKeys optional.Value[*WebMeasurementTestKeys] `json:"test_keys"`
}

// WebMeasurementTestKeys is the canonical container for observations
// generated by most OONI experiments. This is the data format ingested
// by this package for generating [*WebEndpointObservations].
//
// This structure is designed to support Web Connectivity LTE and possibly
// other OONI experiments such as telegram, signal, etc.
type WebMeasurementTestKeys struct {
	// Control contains the OPTIONAL TH response.
	Control optional.Value[*model.THResponse] `json:"control"`

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

	// XControlRequest contains the OPTIONAL TH request.
	XControlRequest optional.Value[*model.THRequest] `json:"x_control_request"`
}
