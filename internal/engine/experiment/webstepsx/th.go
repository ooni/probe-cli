package webstepsx

//
// TH (Test Helper)
//
// This file contains an implementation of the
// (proposed) websteps test helper spec.
//

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

//
// Messages exchanged by the TH client and server
//

// THClientRequest is the request received by the test helper.
type THClientRequest struct {
	// Endpoints is a list of endpoints to measure.
	Endpoints []*measurex.Endpoint

	// URL is the URL we want to measure.
	URL string

	// HTTPRequestHeaders contains the request headers.
	HTTPRequestHeaders http.Header
}

// THServerResponse is the response from the test helper.
type THServerResponse = measurex.THMeasurement

// thMaxAcceptableBodySize is the maximum acceptable body size by TH code.
const thMaxAcceptableBodySize = 1 << 20

//
// TH client implementation
//

// THClient is the high-level API to invoke the TH. This API
// should be used by command line clients.
type THClient struct {
	// DNSServers is the MANDATORY list of DNS-over-UDP
	// servers to use to discover endpoints locally.
	DNServers []*measurex.ResolverInfo

	// HTTPClient is the MANDATORY HTTP client to
	// use for contacting the TH.
	HTTPClient model.HTTPClient

	// ServerURL is the MANDATORY URL of the TH HTTP endpoint.
	ServerURL string
}

// Run calls the TH and returns the response or an error.
//
// Arguments:
//
// - ctx is the context with timeout/deadline/cancellation
//
// - URL is the URL the TH server should measure for us
//
// Algorithm:
//
// - use DNSServers to discover extra endpoints for the target URL
//
// - call the TH using the HTTPClient and the ServerURL
//
// - return response or error.
func (c *THClient) Run(ctx context.Context, URL string) (*THServerResponse, error) {
	parsed, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	mx := measurex.NewMeasurerWithDefaultSettings()
	var dns []*measurex.DNSMeasurement
	const parallelism = 3
	for m := range mx.LookupURLHostParallel(ctx, parallelism, parsed, c.DNServers...) {
		dns = append(dns, m)
	}
	endpoints, err := measurex.AllEndpointsForURL(parsed, dns...)
	if err != nil {
		return nil, err
	}
	return (&THClientCall{
		Endpoints:  endpoints,
		HTTPClient: c.HTTPClient,
		Header:     measurex.NewHTTPRequestHeaderForMeasuring(),
		THURL:      c.ServerURL,
		TargetURL:  URL,
	}).Call(ctx)
}

// THClientCall allows to perform a single TH client call. Make sure
// you fill all the fields marked as MANDATORY before use.
type THClientCall struct {
	// Endpoints contains the MANDATORY endpoints we discovered.
	Endpoints []*measurex.Endpoint

	// HTTPClient is the MANDATORY HTTP client to
	// use for contacting the TH.
	HTTPClient model.HTTPClient

	// Header contains the MANDATORY request headers.
	Header http.Header

	// THURL is the MANDATORY test helper URL.
	THURL string

	// TargetURL is the MANDATORY URL to measure.
	TargetURL string

	// UserAgent is the OPTIONAL user-agent to use.
	UserAgent string
}

// Call performs the specified TH call and returns either a response or an error.
func (c *THClientCall) Call(ctx context.Context) (*THServerResponse, error) {
	creq := &THClientRequest{
		Endpoints:          c.Endpoints,
		URL:                c.TargetURL,
		HTTPRequestHeaders: c.Header,
	}
	reqBody, err := json.Marshal(creq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, "POST", c.THURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	return c.httpClientDo(req)
}

// errTHRequestFailed is the error returned if the TH response is not 200 Ok.
var errTHRequestFailed = errors.New("th: request failed")

func (c *THClientCall) httpClientDo(req *http.Request) (*THServerResponse, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 { // THHandler returns either 400 or 200
		return nil, errTHRequestFailed
	}
	r := io.LimitReader(resp.Body, thMaxAcceptableBodySize)
	respBody, err := netxlite.ReadAllContext(req.Context(), r)
	if err != nil {
		return nil, err
	}
	var sresp THServerResponse
	if err := json.Unmarshal(respBody, &sresp); err != nil {
		return nil, err
	}
	return &sresp, nil
}

//
// TH server implementation
//

// THHandler implements the test helper API.
//
// This handler exposes a unique HTTP endpoint that you need to
// mount to the desired path when creating the server.
//
// The canonical mount point for the HTTP endpoint is /api/v1/websteps.
//
// Accepted methods and request body:
//
// - we only accept POST;
//
// - we expect a THClientRequest as the body.
//
// Status code and response body:
//
// - on success, status is 200 and THServerResponse is the body;
//
// - on failure, status is 400 and there is no body.
//
type THHandler struct{}

// ServerHTTP implements http.Handler.ServeHTTP.
func (h *THHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Server", fmt.Sprintf("oohelperd/%s", version.Version))
	if req.Method != "POST" {
		w.WriteHeader(400)
		return
	}
	reader := io.LimitReader(req.Body, thMaxAcceptableBodySize)
	data, err := netxlite.ReadAllContext(req.Context(), reader)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	var creq THClientRequest
	if err := json.Unmarshal(data, &creq); err != nil {
		w.WriteHeader(400)
		return
	}
	cresp, err := h.singleStep(req.Context(), &creq)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	// We assume that the following call cannot fail because it's a
	// clearly serializable data structure.
	data, err = json.Marshal(cresp)
	runtimex.PanicOnError(err, "json.Marshal failed")
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}

// singleStep performs a singleStep measurement.
//
// The function name derives from the definition (we invented)
// of "web steps". Each redirection is a step. For each step you
// need to figure out the endpoints to use with the DNS. After
// that, you need to check all endpoints. Because here we do not
// perform redirection, this is just a single "step".
//
// The algorithm is the following:
//
// 1. parse the URL and return error if it does not parse or
// the scheme is neither HTTP nor HTTPS;
//
// 2. discover additional endpoints using a suitable DoH
// resolver and the URL's hostname as the domain;
//
// 3. measure each discovered endpoint.
//
// The return value is either a THServerResponse or an error.
func (h *THHandler) singleStep(
	ctx context.Context, req *THClientRequest) (*THServerResponse, error) {
	mx := measurex.NewMeasurerWithDefaultSettings()
	mx.MeasureURLHelper = &thMeasureURLHelper{req.Endpoints}
	mx.Resolvers = []*measurex.ResolverInfo{{
		Network:         measurex.ResolverForeign,
		ForeignResolver: thResolver,
	}}
	jar := measurex.NewCookieJar()
	const parallelism = 3
	meas, err := mx.MeasureURL(ctx, parallelism, req.URL, req.HTTPRequestHeaders, jar)
	if err != nil {
		return nil, err
	}
	return &THServerResponse{
		DNS:       meas.DNS,
		Endpoints: h.simplifyEndpoints(meas.Endpoints),
	}, nil
}

func (h *THHandler) simplifyEndpoints(
	in []*measurex.HTTPEndpointMeasurement) (out []*measurex.HTTPEndpointMeasurement) {
	for _, epnt := range in {
		out = append(out, &measurex.HTTPEndpointMeasurement{
			URL:         epnt.URL,
			Network:     epnt.Network,
			Address:     epnt.Address,
			Measurement: h.simplifyMeasurement(epnt.Measurement),
		})
	}
	return
}

func (h *THHandler) simplifyMeasurement(in *measurex.Measurement) (out *measurex.Measurement) {
	out = &measurex.Measurement{
		Connect:        in.Connect,
		TLSHandshake:   h.simplifyHandshake(in.TLSHandshake),
		QUICHandshake:  h.simplifyHandshake(in.QUICHandshake),
		LookupHost:     in.LookupHost,
		LookupHTTPSSvc: in.LookupHTTPSSvc,
		HTTPRoundTrip:  h.simplifyHTTPRoundTrip(in.HTTPRoundTrip),
	}
	return
}

func (h *THHandler) simplifyHandshake(
	in []*measurex.QUICTLSHandshakeEvent) (out []*measurex.QUICTLSHandshakeEvent) {
	for _, ev := range in {
		out = append(out, &measurex.QUICTLSHandshakeEvent{
			CipherSuite:     ev.CipherSuite,
			Failure:         ev.Failure,
			NegotiatedProto: ev.NegotiatedProto,
			TLSVersion:      ev.TLSVersion,
			PeerCerts:       nil,
			Finished:        0,
			RemoteAddr:      ev.RemoteAddr,
			SNI:             ev.SNI,
			ALPN:            ev.ALPN,
			SkipVerify:      ev.SkipVerify,
			Network:         ev.Network,
			Started:         0,
		})
	}
	return
}

func (h *THHandler) simplifyHTTPRoundTrip(
	in []*measurex.HTTPRoundTripEvent) (out []*measurex.HTTPRoundTripEvent) {
	for _, ev := range in {
		out = append(out, &measurex.HTTPRoundTripEvent{
			Failure:                 ev.Failure,
			Method:                  ev.Method,
			URL:                     ev.URL,
			RequestHeaders:          ev.RequestHeaders,
			StatusCode:              ev.StatusCode,
			ResponseHeaders:         ev.ResponseHeaders,
			ResponseBody:            nil, // we don't transfer the body
			ResponseBodyLength:      ev.ResponseBodyLength,
			ResponseBodyIsTruncated: ev.ResponseBodyIsTruncated,
			ResponseBodyIsUTF8:      ev.ResponseBodyIsUTF8,
			Finished:                ev.Finished,
			Started:                 ev.Started,
		})
	}
	return
}

type thMeasureURLHelper struct {
	epnts []*measurex.Endpoint
}

func (thh *thMeasureURLHelper) LookupExtraHTTPEndpoints(
	ctx context.Context, URL *url.URL, headers http.Header,
	serverEpnts ...*measurex.HTTPEndpoint) (
	epnts []*measurex.HTTPEndpoint, thMeaurement *measurex.THMeasurement, err error) {
	for _, epnt := range thh.epnts {
		epnts = append(epnts, &measurex.HTTPEndpoint{
			Domain:  URL.Hostname(),
			Network: epnt.Network,
			Address: epnt.Address,
			SNI:     URL.Hostname(),
			ALPN:    measurex.ALPNForHTTPEndpoint(epnt.Network),
			URL:     URL,
			Header:  headers, // but overriden later anyway
		})
	}
	return
}

// thResolverURL is the DNS resolver URL used by the TH. We use an
// encrypted resolver to reduce the risk that there is DNS-over-UDP
// censorship in the place where we deploy the TH.
const thResolverURL = "https://dns.google/dns-query"

// thResolver is the DNS resolver used by the TH.
//
// Here we're using github.com/apex/log as the logger, which
// is fine because this is backend only code.
var thResolver = netxlite.WrapResolver(log.Log, netxlite.NewSerialResolver(
	netxlite.NewDNSOverHTTPS(http.DefaultClient, thResolverURL),
))
