package websteps

import "net/http"

// Websteps test helper spec messages:

// CtrlRequest is the request sent by the probe to the test helper.
type CtrlRequest struct {
	// URL is the mandatory URL to measure.
	URL string `json:"url"`

	// Headers contains optional headers.
	Headers map[string][]string `json:"headers"`

	// Addrs contains the optional IP addresses resolved by the
	// probe for the domain inside URL.
	Addrs []string `json:"addrs"`
}

// CtrlResponse is the response from the test helper.
type CtrlResponse struct {
	// URLs contains the URLs we should measure. These URLs
	// derive from CtrlRequest.URL.
	URLs []*URLMeasurement `json:"urls"`
}

// URLMeasurement contains all the URLs measured by the test helper.
type URLMeasurement struct {
	// URL is the URL to which this measurement refers.
	URL string `json:"url"`

	// DNS contains the domain names resolved by the test helper.
	DNS *DNSMeasurement `json:"dns"`

	// Endpoints contains endpoint measurements.
	Endpoints []*EndpointMeasurement `json:"endpoints"`

	// RoundTrip is the related round trip. This field MUST NOT be
	// exported as JSON, since it's only used internally by the test
	// helper and it's completely ignored by the probe.
	RoundTrip *RoundTripInfo `json:"-"`
}

// RoundTripInfo contains info on a specific round trip. This data
// structure is not part of the test helper protocol. We use it
// _inside_ the test helper to describe the discovery phase where
// we gather all the URLs that can derive from a given URL.
type RoundTripInfo struct {
	// Proto is the protocol used, it can be "h2", "http/1.1", "h3".
	Proto string

	// Request is the original HTTP request. Headers also include cookies.
	Request *http.Request

	// Response is the HTTP response.
	Response *http.Response

	// SortIndex is the index using for sorting round trips.
	SortIndex int
}

// DNSMeasurement is a DNS measurement.
type DNSMeasurement struct {
	// Domain is the domain we wanted to resolve.
	Domain string `json:"domain"`

	// Failure is the error that occurred.
	Failure *string `json:"failure"`

	// Addrs contains the resolved addresses.
	Addrs []string `json:"addrs"`
}

// EndpointMeasurement is an HTTP measurement where we are using
// a specific TCP/TLS/QUIC endpoint to get the URL.
//
// The specification describes this data structure as the sum of
// three distinct types: HTTPEndpointMeasurement for "http",
// HTTPSEndpointMeasurement for "https", and H3EndpointMeasurement
// for "h3". We don't have sum types here, therefore we use the
// Protocol field to indicate which fields are meaningful.
type EndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string `json:"endpoint"`

	// Protocol is one of "http", "https", and "h3".
	Protocol string `json:"protocol"`

	// TCPConnect is the TCP connect measurement. This field
	// is only meaningful when protocol is "http" or "https."
	TCPConnect *TCPConnectMeasurement `json:"tcp_connect"`

	// QUICHandshake is the QUIC handshake measurement. This field
	// is only meaningful when the protocol is "h3".
	QUICHandshake *QUICHandshakeMeasurement `json:"quic_handshake"`

	// TLSHandshake is the TLS handshake measurement. This field
	// is only meaningful when the protocol is "https".
	TLSHandshake *TLSHandshakeMeasurement `json:"tls_handshake"`

	// HTTPRoundTrip is the related HTTP GET measurement.
	HTTPRoundTrip *HTTPRoundTripMeasurement `json:"http_round_trip"`
}

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

// QUICHandshakeMeasurement is a QUIC handshake measurement.
type QUICHandshakeMeasurement = TLSHandshakeMeasurement

// HTTPRoundTripMeasurement contains a measured HTTP request and
// the corresponding response.
type HTTPRoundTripMeasurement struct {
	// Request contains request data.
	Request *HTTPRequestMeasurement `json:"request"`

	// Response contains response data.
	Response *HTTPResponseMeasurement `json:"response"`
}

// HTTPRequestMeasurement contains request data.
type HTTPRequestMeasurement struct {
	// Method is the request method.
	Method string `json:"method"`

	// URL is the request URL.
	URL string `json:"url"`

	// Headers contains request headers.
	Headers http.Header `json:"headers"`
}

// HTTPResponseMeasurement contains response data.
type HTTPResponseMeasurement struct {
	// BodyLength contains the body length in bytes.
	BodyLength int64 `json:"body_length"`

	// Failure is the error that occurred.
	Failure *string `json:"failure"`

	// Headers contains response headers.
	Headers http.Header `json:"headers"`

	// StatusCode is the response status code.
	StatusCode int64 `json:"status_code"`
}
