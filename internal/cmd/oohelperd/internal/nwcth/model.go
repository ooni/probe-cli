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
	IsEndpointMeasurement()
}

// HTTPEndpointMeasurement is a measurement of a specific HTTP endpoint.
type HTTPEndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string

	// TCPConnectMeasurement is the related TCP connect measurement.
	TCPConnectMeasurement *TCPConnectMeasurement

	// HTTPRoundtripMeasurement is the related HTTP GET measurement.
	HTTPRoundtripMeasurement *HTTPRoundtripMeasurement
}

func (h *HTTPEndpointMeasurement) IsEndpointMeasurement() {}

// HTTPEndpointMeasurement is a measurement of a specific HTTP endpoint.
type HTTPSEndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string

	// TCPConnectMeasurement is the related TCP connect measurement.
	TCPConnectMeasurement *TCPConnectMeasurement

	// TLSHandshakeMeasurement is the related TLS handshake measurement.
	TLSHandshakeMeasurement *TLSHandshakeMeasurement

	// HTTPRequestMeasurement is the related HTTP GET measurement.
	HTTPRoundtripMeasurement *HTTPRoundtripMeasurement
}

func (h *HTTPSEndpointMeasurement) IsEndpointMeasurement() {}

// HTTPEndpointMeasurement is a measurement of a specific HTTP endpoint.
type H3EndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string

	// QUICHandshakeMeasurement is the related QUIC(TLS 1.3) handshake measurement.
	QUICHandshakeMeasurement *TLSHandshakeMeasurement

	// HTTPRequestMeasurement is the related HTTP GET measurement.
	HTTPRoundtripMeasurement *HTTPRoundtripMeasurement
}

func (h *H3EndpointMeasurement) IsEndpointMeasurement() {}

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

type HTTPRoundtripMeasurement struct {
	Request  *HTTPRequest
	Response *HTTPResponse
}

// HTTPRequestMeasurement is a HTTP request measurement.
type HTTPRequest struct {
	Headers http.Header
}

// HTTPRequestMeasurement is a HTTP request measurement.
type HTTPResponse struct {
	BodyLength int64       `json:"body_length"`
	Failure    *string     `json:"failure"`
	Headers    http.Header `json:"headers"`
	StatusCode int64       `json:"status_code"`
}
