package nwcth

import "net/http"

// URLMeasurement is a measurement of a given URL that
// includes connectivity measurement for each endpoint
// implied by the given URL.
type URLMeasurement struct {
	// URL is the URL we're using
	URL string `json:"url"`

	// DNS contains the domain names resolved by the helper.
	DNS *DNSMeasurement `json:"dns"`

	// RoundTrip is the related round trip.
	RoundTrip *RoundTrip `json:"-"`

	// Endpoints contains endpoint measurements.
	Endpoints []EndpointMeasurement `json:"endpoints"`
}

// DNSMeasurement is a DNS measurement.
type DNSMeasurement struct {
	// Domain is the domain we wanted to resolve.
	Domain string `json:"domain"`

	// Addrs contains the resolved addresses.
	Addrs []string `json:"addrs"`

	// Failure is the error that occurred.
	Failure *string `json:"failure"`
}

type EndpointMeasurement interface {
	GetHTTPRoundTrip() *HTTPRoundtripMeasurement
}

// HTTPEndpointMeasurement is the measurement of requesting a specific endpoint via HTTP.
type HTTPEndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string `json:"endpoint"`

	// Protocol is http
	Protocol string `json:"protocol"`

	// TCPConnectMeasurement is the related TCP connect measurement.
	TCPConnectMeasurement *TCPConnectMeasurement `json:"tcp_connect"`

	// HTTPRoundtripMeasurement is the related HTTP GET measurement.
	HTTPRoundtripMeasurement *HTTPRoundtripMeasurement `json:"http_round_trip"`
}

func (h *HTTPEndpointMeasurement) GetHTTPRoundTrip() *HTTPRoundtripMeasurement {
	return h.HTTPRoundtripMeasurement
}

// HTTPSEndpointMeasurement is the measurement of requesting a specific endpoint via HTTPS.
type HTTPSEndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string `json:"endpoint"`

	// Protocol is https
	Protocol string `json:"protocol"`

	// TCPConnectMeasurement is the related TCP connect measurement.
	TCPConnectMeasurement *TCPConnectMeasurement `json:"tcp_connect"`

	// TLSHandshakeMeasurement is the related TLS handshake measurement.
	TLSHandshakeMeasurement *TLSHandshakeMeasurement `json:"tls_handshake"`

	// HTTPRoundtripMeasurement is the related HTTP GET measurement.
	HTTPRoundtripMeasurement *HTTPRoundtripMeasurement `json:"http_round_trip"`
}

func (h *HTTPSEndpointMeasurement) GetHTTPRoundTrip() *HTTPRoundtripMeasurement {
	return h.HTTPRoundtripMeasurement
}

// H3EndpointMeasurement is the measurement of requesting a specific endpoint via HTTP/3.
type H3EndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string `json:"endpoint"`

	// Protocol is h3, or h3-29
	Protocol string `json:"protocol"`

	// QUICHandshakeMeasurement is the related QUIC(TLS 1.3) handshake measurement.
	QUICHandshakeMeasurement *TLSHandshakeMeasurement `json:"quic_handshake"`

	// HTTPRoundtripMeasurement is the related HTTP GET measurement.
	HTTPRoundtripMeasurement *HTTPRoundtripMeasurement `json:"http_round_trip"`
}

func (h *H3EndpointMeasurement) GetHTTPRoundTrip() *HTTPRoundtripMeasurement {
	return h.HTTPRoundtripMeasurement
}

// Implementation note: OONI uses nil to indicate no error but here
// it's more convenient to just use an empty string.

// TCPConnectMeasurement is a TCP connect measurement.
type TCPConnectMeasurement struct {
	// Failure is the error that occurred.
	Failure *string `json:"failure"`
}

// TLSHandshakeMeasurement is a TLS handshake measurement.
type TLSHandshakeMeasurement struct {
	// Failure is the error that occurred.
	Failure *string `json:"failure"`
}

// HTTPRoundtripMeasurement contains a measured HTTP request and the corresponding response.
type HTTPRoundtripMeasurement struct {
	Request  *HTTPRequest  `json:"request"`
	Response *HTTPResponse `json:"response"`
}

// HTTPRequest contains the headers of the measured HTTP Get request.
type HTTPRequest struct {
	Headers http.Header `json:"headers"`
	Method  string      `json:"method"`
	URL     string      `json:"url"`
}

// HTTPResponse contains the response of the measured HTTP Get request.
type HTTPResponse struct {
	BodyLength int64       `json:"body_length"`
	Failure    *string     `json:"failure"`
	Headers    http.Header `json:"headers"`
	StatusCode int64       `json:"status_code"`
}
