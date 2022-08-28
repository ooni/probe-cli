package webconnectivity

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TODO(bassosimone): these struct definitions should be moved outside the
// specific implementation of Web Connectivity v0.4.

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

// ControlTLSHandshakeResult is the result of the TLS handshake
// attempt performed by the control vantage point.
type ControlTLSHandshakeResult struct {
	ServerName string  `json:"server_name"`
	Status     bool    `json:"status"`
	Failure    *string `json:"failure"`
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

// TODO(bassosimone): ASNs and FillASNs are private implementation details of v0.4
// that are actually ~annoying because we are mixing the data model with fields seen
// by just the v0.4 client implementation. We should avoid repeating this mistake
// when implementing v0.5 of the client.

// ControlDNSResult is the result of the DNS lookup
// performed by the control vantage point.
type ControlDNSResult struct {
	Failure *string  `json:"failure"`
	Addrs   []string `json:"addrs"`
	ASNs    []int64  `json:"-"` // not visible from the JSON
}

// ControlIPInfo contains information about IP addresses resolved either
// by the probe or by the TH and processed by the TH.
type ControlIPInfo struct {
	// ASN contains the address' AS number.
	ASN int64 `json:"asn"`

	// Flags contains flags describing this address.
	Flags int64 `json:"flags"`
}

const (
	// ControlIPInfoFlagResolvedByProbe indicates that the probe has
	// resolved this IP address.
	ControlIPInfoFlagResolvedByProbe = 1 << iota

	// ControlIPInfoFlagResolvedByTH indicates that the test helper
	// has resolved this IP address.
	ControlIPInfoFlagResolvedByTH

	// ControlIPInfoFlagIsBogon indicates that the address is a bogon
	ControlIPInfoFlagIsBogon
)

// ControlResponse is the response from the control service.
type ControlResponse struct {
	TCPConnect   map[string]ControlTCPConnectResult   `json:"tcp_connect"`
	TLSHandshake map[string]ControlTLSHandshakeResult `json:"tls_handshake"`
	HTTPRequest  ControlHTTPRequestResult             `json:"http_request"`
	DNS          ControlDNSResult                     `json:"dns"`
	IPInfo       map[string]ControlIPInfo             `json:"ip_info"`
}

// Control performs the control request and returns the response.
func Control(
	ctx context.Context, sess model.ExperimentSession,
	thAddr string, creq ControlRequest) (out ControlResponse, err error) {
	clnt := &httpx.APIClientTemplate{
		BaseURL:    thAddr,
		HTTPClient: sess.DefaultHTTPClient(),
		Logger:     sess.Logger(),
		UserAgent:  sess.UserAgent(),
	}
	sess.Logger().Infof("control for %s...", creq.HTTPRequest)
	// make sure error is wrapped
	err = clnt.WithBodyLogging().Build().PostJSON(ctx, "/", creq, &out)
	if err != nil {
		err = netxlite.NewTopLevelGenericErrWrapper(err)
	}
	sess.Logger().Infof("control for %s... %+v", creq.HTTPRequest, model.ErrorToStringOrOK(err))
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
