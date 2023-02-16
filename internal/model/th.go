package model

// THDNSNameError is the error returned by the control on NXDOMAIN
const THDNSNameError = "dns_name_error"

// THRequest is the request that we send to the control.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-017-web-connectivity.md
type THRequest struct {
	HTTPRequest        string              `json:"http_request"`
	HTTPRequestHeaders map[string][]string `json:"http_request_headers"`
	TCPConnect         []string            `json:"tcp_connect"`

	// XQUICEnabled is a feature flag that tells the oohelperd to
	// conditionally enable QUIC measurements, which are otherwise
	// disabled by default. We will honour this flag during the
	// v3.17.x release cycle and possibly also for v3.18.x but we
	// will eventually enable QUIC for all clients.
	XQUICEnabled bool `json:"x_quic_enabled"`
}

// THTCPConnectResult is the result of the TCP connect
// attempt performed by the control vantage point.
type THTCPConnectResult struct {
	Status  bool    `json:"status"`
	Failure *string `json:"failure"`
}

// THTLSHandshakeResult is the result of the TLS handshake
// attempt performed by the control vantage point.
type THTLSHandshakeResult struct {
	ServerName string  `json:"server_name"`
	Status     bool    `json:"status"`
	Failure    *string `json:"failure"`
}

// THHTTPRequestResult is the result of the HTTP request
// performed by the control vantage point.
type THHTTPRequestResult struct {
	BodyLength           int64             `json:"body_length"`
	DiscoveredH3Endpoint string            `json:"discovered_h3_endpoint"`
	Failure              *string           `json:"failure"`
	Title                string            `json:"title"`
	Headers              map[string]string `json:"headers"`
	StatusCode           int64             `json:"status_code"`
}

// TODO(bassosimone): ASNs is a private implementation detail of v0.4
// that is actually ~annoying because we are mixing the data model with fields used
// by just the v0.4 client implementation. We should avoid repeating this mistake
// when implementing v0.5 of the client and eventually remove ASNs.

// THDNSResult is the result of the DNS lookup
// performed by the control vantage point.
type THDNSResult struct {
	Failure *string  `json:"failure"`
	Addrs   []string `json:"addrs"`
	ASNs    []int64  `json:"-"` // not visible from the JSON
}

// THIPInfo contains information about IP addresses resolved either
// by the probe or by the TH and processed by the TH.
type THIPInfo struct {
	// ASN contains the address' AS number.
	ASN int64 `json:"asn"`

	// Flags contains flags describing this address.
	Flags int64 `json:"flags"`
}

const (
	// THIPInfoFlagResolvedByProbe indicates that the probe has
	// resolved this IP address.
	THIPInfoFlagResolvedByProbe = 1 << iota

	// THIPInfoFlagResolvedByTH indicates that the test helper
	// has resolved this IP address.
	THIPInfoFlagResolvedByTH

	// THIPInfoFlagIsBogon indicates that the address is a bogon
	THIPInfoFlagIsBogon

	// THIPInfoFlagValidForDomain indicates that an IP address
	// is valid for the domain because it works with TLS
	THIPInfoFlagValidForDomain
)

// THResponse is the response from the control service.
type THResponse struct {
	TCPConnect    map[string]THTCPConnectResult   `json:"tcp_connect"`
	TLSHandshake  map[string]THTLSHandshakeResult `json:"tls_handshake,omitempty"`
	QUICHandshake map[string]THTLSHandshakeResult `json:"quic_handshake"`
	HTTPRequest   THHTTPRequestResult             `json:"http_request"`
	HTTP3Request  *THHTTPRequestResult            `json:"http3_request"` // optional!
	DNS           THDNSResult                     `json:"dns"`
	IPInfo        map[string]*THIPInfo            `json:"ip_info,omitempty"`
}
