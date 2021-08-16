package websteps

import "net/http"

// RoundTrip describes a specific round trip.
type RoundTrip struct {
	// proto is the protocol used, it can be "h2", "http/1.1", "h3", "h3-29"
	Proto string

	// Request is the original HTTP request. The headers
	// also include cookies.
	Request *http.Request

	// Response is the HTTP response.
	Response *http.Response

	// sortIndex is an internal field using for sorting.
	SortIndex int
}

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
	Endpoints []*EndpointMeasurement `json:"endpoints"`
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

// HTTPSEndpointMeasurement is the measurement of requesting a specific endpoint via HTTPS.
type EndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string `json:"endpoint"`

	// Protocol is the used protocol. It can be "http", "https", "h3", "h3-29" or other supported QUIC protocols
	Protocol string `json:"protocol"`

	// TCPConnectMeasurement is the related TCP connect measurement, if applicable (nil for h3 requests)
	TCPConnectMeasurement *TCPConnectMeasurement `json:"tcp_connect"`

	// QUICHandshakeMeasurement is the related QUIC(TLS 1.3) handshake measurement, if applicable (nil for http, https requests)
	QUICHandshakeMeasurement *TLSHandshakeMeasurement `json:"quic_handshake"`

	// TLSHandshakeMeasurement is the related TLS handshake measurement, if applicable (nil for http, h3 requests)
	TLSHandshakeMeasurement *TLSHandshakeMeasurement `json:"tls_handshake"`

	// HTTPRoundTripMeasurement is the related HTTP GET measurement.
	HTTPRoundTripMeasurement *HTTPRoundTripMeasurement `json:"http_round_trip"`
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

// HTTPRoundTripMeasurement contains a measured HTTP request and the corresponding response.
type HTTPRoundTripMeasurement struct {
	Request  *HTTPRequestMeasurement  `json:"request"`
	Response *HTTPResponseMeasurement `json:"response"`
}

// HTTPRequestMeasurement contains the headers of the measured HTTP Get request.
type HTTPRequestMeasurement struct {
	Headers http.Header `json:"headers"`
	Method  string      `json:"method"`
	URL     string      `json:"url"`
}

// HTTPResponseMeasurement contains the response of the measured HTTP Get request.
type HTTPResponseMeasurement struct {
	BodyLength int64       `json:"body_length"`
	Failure    *string     `json:"failure"`
	Headers    http.Header `json:"headers"`
	StatusCode int64       `json:"status_code"`
}
