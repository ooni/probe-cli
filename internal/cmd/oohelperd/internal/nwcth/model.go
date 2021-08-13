package nwcth

import "net/http"

// URLMeasurement is a measurement of a given URL that
// includes connectivity measurement for each endpoint
// implied by the given URL.
type URLMeasurement struct {
	// URL is the URL we're using
	URL string

	// DNS contains the domain names resolved by the helper.
	DNS *DNSMeasurement

	// RoundTrip is the related round trip.
	RoundTrip *RoundTrip `json:"-"`

	// Endpoints contains endpoint measurements.
	Endpoints []EndpointMeasurement
}

// DNSMeasurement is a DNS measurement.
type DNSMeasurement struct {
	// Domain is the domain we wanted to resolve.
	Domain string

	// Addrs contains the resolved addresses.
	Addrs []string

	// Failure is the error that occurred.
	Failure *string
}

type EndpointMeasurement interface {
	GetHTTPRoundTrip() *HTTPRoundtripMeasurement
}

// HTTPEndpointMeasurement is the measurement of requesting a specific endpoint via HTTP.
type HTTPEndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string

	// Protocol is http
	Protocol string

	// TCPConnectMeasurement is the related TCP connect measurement.
	TCPConnectMeasurement *TCPConnectMeasurement

	// HTTPRoundtripMeasurement is the related HTTP GET measurement.
	HTTPRoundtripMeasurement *HTTPRoundtripMeasurement
}

func (h *HTTPEndpointMeasurement) GetHTTPRoundTrip() *HTTPRoundtripMeasurement {
	return h.HTTPRoundtripMeasurement
}

// HTTPSEndpointMeasurement is the measurement of requesting a specific endpoint via HTTPS.
type HTTPSEndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string

	// Protocol is https
	Protocol string

	// TCPConnectMeasurement is the related TCP connect measurement.
	TCPConnectMeasurement *TCPConnectMeasurement

	// TLSHandshakeMeasurement is the related TLS handshake measurement.
	TLSHandshakeMeasurement *TLSHandshakeMeasurement

	// HTTPRoundtripMeasurement is the related HTTP GET measurement.
	HTTPRoundtripMeasurement *HTTPRoundtripMeasurement
}

func (h *HTTPSEndpointMeasurement) GetHTTPRoundTrip() *HTTPRoundtripMeasurement {
	return h.HTTPRoundtripMeasurement
}

// H3EndpointMeasurement is the measurement of requesting a specific endpoint via HTTP/3.
type H3EndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string

	// Protocol is h3, or h3-29
	Protocol string

	// QUICHandshakeMeasurement is the related QUIC(TLS 1.3) handshake measurement.
	QUICHandshakeMeasurement *TLSHandshakeMeasurement

	// HTTPRoundtripMeasurement is the related HTTP GET measurement.
	HTTPRoundtripMeasurement *HTTPRoundtripMeasurement
}

func (h *H3EndpointMeasurement) GetHTTPRoundTrip() *HTTPRoundtripMeasurement {
	return h.HTTPRoundtripMeasurement
}

// Implementation note: OONI uses nil to indicate no error but here
// it's more convenient to just use an empty string.

// TCPConnectMeasurement is a TCP connect measurement.
type TCPConnectMeasurement struct {
	// Failure is the error that occurred.
	Failure *string
}

// TLSHandshakeMeasurement is a TLS handshake measurement.
type TLSHandshakeMeasurement struct {
	// Failure is the error that occurred.
	Failure *string
}

// HTTPRoundtripMeasurement contains a measured HTTP request and the corresponding response.
type HTTPRoundtripMeasurement struct {
	Request  *HTTPRequest
	Response *HTTPResponse
}

// HTTPRequest contains the headers of the measured HTTP Get request.
type HTTPRequest struct {
	Headers http.Header
}

// HTTPResponse contains the response of the measured HTTP Get request.
type HTTPResponse struct {
	BodyLength int64       `json:"body_length"`
	Failure    *string     `json:"failure"`
	Headers    http.Header `json:"headers"`
	StatusCode int64       `json:"status_code"`
}
