package nwebconnectivity

import (
	"context"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/engine/httpx"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
)

// ControlRequest is the request that we send to the control
type ControlRequest struct {
	HTTPRequest        string              `json:"http_request"`
	HTTPRequestHeaders map[string][]string `json:"http_request_headers"`
	TCPConnect         []string            `json:"tcp_connect"`
}

// ControlResponse is the response from the control service.
type ControlResponse struct {
	URLMeasurements []*ControlURLMeasurement `json:"urls"`
}

// URLMeasurement is the result of resolving and requesting a URL
type ControlURLMeasurement struct {
	URL       string                       `json:"url"`
	DNS       *ControlDNSMeasurement       `json:"dns"`
	Endpoints []ControlEndpointMeasurement `json:"endpoints"`
}

// ControlDNSMeasurement is the result of the DNS lookup
// performed by the control vantage point.
type ControlDNSMeasurement struct {
	Failure *string  `json:"failure"`
	Addrs   []string `json:"addrs"`
}

// ControlEndpointMeasurement is the sum of ControlHTTPMeasurement and ControlH3Measurement
type ControlEndpointMeasurement interface {
	GetHTTPRequestMeasurement() *ControlHTTPRequestMeasurement
}

// ControlHTTPMeasurement is the TCP transport and application layer HTTP(S) result
// performed by the control vantage point.
type ControlHTTPMeasurement struct {
	Endpoint     string                          `json:"endpoint"`
	Protocol     string                          `json:"protocol"`
	TCPConnect   *ControlTCPConnect              `json:"tcp_connect"`
	TLSHandshake *ControlTLSHandshakeMeasurement `json:"tls_handshake"`
	HTTPRequest  *ControlHTTPRequestMeasurement  `json:"http_request"`
}

func (h *ControlHTTPMeasurement) GetHTTPRequestMeasurement() *ControlHTTPRequestMeasurement {
	return h.HTTPRequest
}

// ControlH3Measurement is the QUIC transport and application layer HTTP/3 result
// performed by the control vantage point.
type ControlH3Measurement struct {
	Endpoint      string                          `json:"endpoint"`
	Protocol      string                          `json:"protocol"`
	QUICHandshake *ControlTLSHandshakeMeasurement `json:"quic_handshake"`
	HTTPRequest   *ControlHTTPRequestMeasurement  `json:"http_request"`
}

func (h *ControlH3Measurement) GetHTTPRequestMeasurement() *ControlHTTPRequestMeasurement {
	return h.HTTPRequest
}

// ControlTCPConnect is the result of the TCP connect
// attempt performed by the control vantage point.
type ControlTCPConnect struct {
	Failure *string `json:"failure"`
}

// ControlTLSHandshakeMeasurement is the result of the TLS handshake
// attempt performed by the control vantage point.
type ControlTLSHandshakeMeasurement struct {
	Failure *string `json:"failure"`
}

// ControlHTTPRequestMeasurement is the result of the HTTP request
// performed by the control vantage point.
type ControlHTTPRequestMeasurement struct {
	BodyLength int64       `json:"body_length"`
	Failure    *string     `json:"failure"`
	Headers    http.Header `json:"headers"`
	StatusCode int64       `json:"status_code"`
}

// Control performs the control request and returns the response.
func Control(
	ctx context.Context, sess model.ExperimentSession,
	thAddr string, creq ControlRequest) (out ControlResponse, err error) {
	clnt := httpx.Client{
		BaseURL:    thAddr,
		HTTPClient: sess.DefaultHTTPClient(),
		Logger:     sess.Logger(),
	}
	// make sure error is wrapped
	err = errorsx.SafeErrWrapperBuilder{
		Error:     clnt.PostJSON(ctx, "/", creq, &out),
		Operation: errorsx.TopLevelOperation,
	}.MaybeBuild()
	return
}

func findTestHelper(e model.ExperimentSession) (testhelper *model.Service) {
	testhelpers, _ := e.GetTestHelpersByName("web-connectivity")
	for _, th := range testhelpers {
		if th.Type == "https" {
			testhelper = &th
			break
		}
	}
	return testhelper
}
