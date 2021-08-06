package nwebconnectivity

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/httpx"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
)

// NextLocationInfo contains the redirected location as well as the request object which forwards most headers of the initial request.
// This forwarded request is generated by the http.Client and
type NextLocationInfo struct {
	Jar             http.CookieJar
	Location        *url.URL
	HTTPRedirectReq *http.Request
}

// ControlRequest is the request that we send to the control
type ControlRequest struct {
	HTTPCookieJar      http.CookieJar      `json:"-"`
	HTTPRequest        string              `json:"http_request"`
	HTTPRequestHeaders map[string][]string `json:"http_request_headers"`
	TCPConnect         []string            `json:"tcp_connect"`
}

// ControlResponse is the response from the control service.
type ControlResponse struct {
	URLMeasurements []*ControlURL `json:"urls"`
}

// URLMeasurement is the result of resolving and requesting a URL
type ControlURL struct {
	URL       string                 `json:"url"`
	DNS       *ControlDNSMeasurement `json:"dns"`
	Endpoints []ControlEndpoint      `json:"endpoints"`
}

// ControlDNSMeasurement is the result of the DNS lookup
// performed by the control vantage point.
type ControlDNSMeasurement struct {
	Failure *string  `json:"failure"`
	Addrs   []string `json:"addrs"`
}

// ControlEndpoint is the sum of ControlHTTP and ControlH3
type ControlEndpoint interface {
	IsEndpointMeasurement()
}

// ControlHTTP is the TCP transport and application layer HTTP(S) result
// performed by the control vantage point.
type ControlHTTP struct {
	Endpoint     string               `json:"endpoint"`
	Protocol     string               `json:"protocol"`
	TCPConnect   *ControlTCPConnect   `json:"tcp_connect"`
	TLSHandshake *ControlTLSHandshake `json:"tls_handshake"`
	HTTPRequest  *ControlHTTPRequest  `json:"http_request"`
}

func (h *ControlHTTP) IsEndpointMeasurement() {}

// ControlH3 is the QUIC transport and application layer HTTP/3 result
// performed by the control vantage point.
type ControlH3 struct {
	Endpoint      string               `json:"endpoint"`
	Protocol      string               `json:"protocol"`
	QUICHandshake *ControlTLSHandshake `json:"quic_handshake"`
	HTTPRequest   *ControlHTTPRequest  `json:"http_request"`
}

func (h *ControlH3) IsEndpointMeasurement() {}

// ControlTCPConnect is the result of the TCP connect
// attempt performed by the control vantage point.
type ControlTCPConnect struct {
	Failure *string `json:"failure"`
}

// ControlTLSHandshake is the result of the TLS handshake
// attempt performed by the control vantage point.
type ControlTLSHandshake struct {
	Failure *string `json:"failure"`
}

// ControlHTTPRequest is the result of the HTTP request
// performed by the control vantage point.
type ControlHTTPRequest struct {
	BodyLength int64             `json:"body_length"`
	Failure    *string           `json:"failure"`
	Headers    map[string]string `json:"headers"`
	StatusCode int64             `json:"status_code"`
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
