package webconnectivity

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/engine/httpx"
	legacyerrorsx "github.com/ooni/probe-cli/v3/internal/engine/legacy/errorsx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// ControlRequest is the request that we send to the control
type ControlRequest struct {
	HTTPRequest        string              `json:"http_request"`
	HTTPRequestHeaders map[string][]string `json:"http_request_headers"`
	TCPConnect         []string            `json:"tcp_connect"`
}

// ControlTCPConnectResult is the result of the TCP connect
// attempt performed by the control vantage point.
type ControlTCPConnectResult struct {
	Status  bool    `json:"status"`
	Failure *string `json:"failure"`
}

// ControlHTTPRequestResult is the result of the HTTP request
// performed by the control vantage point.
type ControlHTTPRequestResult struct {
	BodyLength int64             `json:"body_length"`
	Failure    *string           `json:"failure"`
	Title      string            `json:"title"`
	Headers    map[string]string `json:"headers"`
	StatusCode int64             `json:"status_code"`
}

// ControlDNSResult is the result of the DNS lookup
// performed by the control vantage point.
type ControlDNSResult struct {
	Failure *string  `json:"failure"`
	Addrs   []string `json:"addrs"`
	ASNs    []int64  `json:"-"` // not visible from the JSON
}

// ControlResponse is the response from the control service.
type ControlResponse struct {
	TCPConnect  map[string]ControlTCPConnectResult `json:"tcp_connect"`
	HTTPRequest ControlHTTPRequestResult           `json:"http_request"`
	DNS         ControlDNSResult                   `json:"dns"`
}

// Control performs the control request and returns the response.
func Control(
	ctx context.Context, sess model.ExperimentSession,
	thAddr string, creq ControlRequest) (out ControlResponse, err error) {
	clnt := httpx.Client{
		BaseURL:    thAddr,
		HTTPClient: sess.DefaultHTTPClient(),
		Logger:     sess.Logger(),
		UserAgent:  sess.UserAgent(),
	}
	sess.Logger().Infof("control for %s...", creq.HTTPRequest)
	// make sure error is wrapped
	err = legacyerrorsx.SafeErrWrapperBuilder{
		Error:     clnt.PostJSON(ctx, "/", creq, &out),
		Operation: netxlite.TopLevelOperation,
	}.MaybeBuild()
	sess.Logger().Infof("control for %s... %+v", creq.HTTPRequest, err)
	(&out.DNS).FillASNs(sess)
	return
}

// FillASNs fills the ASNs array of ControlDNSResult. For each Addr inside
// of the ControlDNSResult structure, we obtain the corresponding ASN.
//
// This is very useful to know what ASNs were the IP addresses returned by
// the control according to the probe's ASN database.
func (dns *ControlDNSResult) FillASNs(sess model.ExperimentSession) {
	dns.ASNs = []int64{}
	for _, ip := range dns.Addrs {
		// TODO(bassosimone): this would be more efficient if we'd open just
		// once the database and then reuse it for every address.
		asn, _, _ := geolocate.LookupASN(ip)
		dns.ASNs = append(dns.ASNs, int64(asn))
	}
}
