package measurex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

//
// Messages exchanged by the TH client and server
//

// THClientRequest is the request received by the test helper.
type THClientRequest struct {
	// Endpoints is a list of endpoints to measure.
	Endpoints []*Endpoint

	// URL is the URL we want to measure.
	URL string

	// HTTPRequestHeaders contains the request headers.
	HTTPRequestHeaders http.Header
}

// THServerResponse is the response from the test helper.
type THServerResponse struct {
	// DNS contains all the DNS related measurements.
	DNS *THDNSMeasurement

	// Endpoints contains a measurement for each endpoint
	// that was discovered by the probe or the TH.
	Endpoints []*THEndpointMeasurement
}

// THDNSMeasurement is a DNS measurement performed by the test helper.
type THDNSMeasurement struct {
	// Oddities lists all the oddities inside this measurement.
	Oddities []Oddity

	// LookupHost contains all the host lookups.
	LookupHost []*THLookupHostEvent `json:",omitempty"`

	// LookupHTTPSSvc contains all the HTTPSSvc lookups.
	LookupHTTPSSvc []*THLookupHTTPSSvcEvent `json:",omitempty"`
}

// THLookupHostEvent is the LookupHost event sent
// back by the test helper.
type THLookupHostEvent struct {
	Network string
	Address string
	Domain  string
	Error   *string
	Oddity  Oddity
	Addrs   []string
}

// THLookupHTTPSSvcEvent is the LookupHTTPSvc event sent
// back by the test helper.
type THLookupHTTPSSvcEvent struct {
	Network string
	Address string
	Domain  string
	Error   *string
	Oddity  Oddity
	IPv4    []string
	IPv6    []string
	ALPN    []string
}

// THEndpointMeasurement is an endpoint measurement
// performed by the test helper.
type THEndpointMeasurement struct {
	// Oddities lists all the oddities inside this measurement.
	Oddities []Oddity

	// Connect contains all the connect operations.
	Connect []*THConnectEvent `json:",omitempty"`

	// TLSHandshake contains all the TLS handshakes.
	TLSHandshake []*THHandshakeEvent `json:",omitempty"`

	// QUICHandshake contains all the QUIC handshakes.
	QUICHandshake []*THHandshakeEvent `json:",omitempty"`

	// HTTPRoundTrip contains all the HTTP round trips.
	HTTPRoundTrip []*THHTTPRoundTripEvent `json:",omitempty"`
}

// THConnectEvent is the connect event sent back by the test helper.
type THConnectEvent struct {
	Network    string
	RemoteAddr string
	Error      *string
	Oddity     Oddity
}

// THHandshakeEvent is the handshake event sent
// back by the test helper.
type THHandshakeEvent struct {
	Network         string
	RemoteAddr      string
	SNI             string
	ALPN            []string
	Error           *string
	Oddity          Oddity
	TLSVersion      string
	CipherSuite     string
	NegotiatedProto string
}

// THHTTPRoundTripEvent is the HTTP round trip event
// sent back by the test helper.
type THHTTPRoundTripEvent struct {
	RequestMethod            string
	RequestURL               string
	RequestHeader            http.Header
	Error                    *string
	Oddity                   Oddity
	ResponseStatus           int64
	ResponseHeader           http.Header
	ResponseBodySnapshotSize int64
	MaxBodySnapshotSize      int64
}

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
	DNServers []string

	// HTTPClient is the MANDATORY HTTP client to
	// use for contacting the TH.
	HTTPClient HTTPClient

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
	mx := NewMeasurerWithDefaultSettings()
	mx.RegisterUDPResolvers(c.DNServers...)
	mx.LookupURLHostParallel(ctx, parsed)
	httpEndpoints, err := mx.DB.SelectAllHTTPEndpointsForURL(parsed)
	if err != nil {
		return nil, err
	}
	var endpoints []*Endpoint
	for _, epnt := range httpEndpoints {
		endpoints = append(endpoints, &Endpoint{
			Network: epnt.Network,
			Address: epnt.Address,
		})
	}
	return (&THClientCall{
		Endpoints:  endpoints,
		HTTPClient: c.HTTPClient,
		Header:     NewHTTPRequestHeaderForMeasuring(),
		THURL:      c.ServerURL,
		TargetURL:  URL,
	}).Call(ctx)
}

// THClientCall allows to perform a single TH client call. Make sure
// you fill all the fields marked as MANDATORY before use.
//
// This is a low-level API for calling the TH. If you are writing
// a CLI client, use THClient. If you are writing code for the
// Measurer, use THMeasurerClientCall.
type THClientCall struct {
	// Endpoints contains the MANDATORY endpoints we discovered.
	Endpoints []*Endpoint

	// HTTPClient is the MANDATORY HTTP client to
	// use for contacting the TH.
	HTTPClient HTTPClient

	// Header contains the MANDATORY request headers.
	Header http.Header

	// THURL is the MANDATORY test helper URL.
	THURL string

	// TargetURL is the MANDATORY URL to measure.
	TargetURL string
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
	req.Header.Set("User-Agent", fmt.Sprintf("ooniprobe-cli/%s", version.Version))
	return c.httpClientDo(req)
}

// errTHRequestFailed is the error returned if the TH response is not 200 Ok.
var errTHRequestFailed = errors.New("th: request failed")

func (c *THClientCall) httpClientDo(req *http.Request) (*THServerResponse, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 { // THHandler returns either 400 or 200
		return nil, errTHRequestFailed
	}
	defer resp.Body.Close()
	r := io.LimitReader(resp.Body, thMaxAcceptableBodySize)
	respBody, err := iox.ReadAllContext(req.Context(), r)
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
	data, err := iox.ReadAllContext(req.Context(), reader)
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
	parsedURL, err := url.Parse(req.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, errors.New("invalid request url")
	}
	epnts, dns := h.dohQuery(ctx, parsedURL)
	m := &THServerResponse{DNS: dns}
	epnts = h.prepareEnpoints(
		epnts, parsedURL, req.Endpoints, req.HTTPRequestHeaders)
	mx := NewMeasurerWithDefaultSettings()
	jar := NewCookieJar()
	for me := range mx.HTTPEndpointGetParallel(ctx, jar, epnts...) {
		m.Endpoints = append(m.Endpoints, h.newTHEndpointMeasurement(me))
	}
	h.maybeQUICFollowUp(ctx, m, epnts...)
	return m, nil
}

// prepareEnpoints takes in input a list of endpoints discovered
// so far by the TH and extends this list by adding the endpoints
// discovered by the client. Before returning, this function
// ensures that we don't have any duplicate endpoint.
//
// Arguments:
//
// - the list of endpoints discovered by the TH
//
// - the URL provided by the probe
//
// - the endpoints provided by the probe
//
// - the headers provided by the probe
//
// The return value may be an empty list if both the client
// and the TH failed to discover any endpoint.
//
// When the return value contains endpoints, we also fill
// the HTTPEndpoint.Header field using the header param
// provided by the client. We don't allow arbitrary headers:
// we only copy a subset of allowed headers.
func (h *THHandler) prepareEnpoints(epnts []*HTTPEndpoint, URL *url.URL,
	clientEpnts []*Endpoint, header http.Header) (out []*HTTPEndpoint) {
	for _, epnt := range clientEpnts {
		epnts = append(epnts, &HTTPEndpoint{
			Domain:  URL.Hostname(),
			Network: epnt.Network,
			Address: epnt.Address,
			SNI:     URL.Hostname(),
			ALPN:    alpnForHTTPEndpoint(epnt.Network),
			URL:     URL,
			Header:  http.Header{}, // see the loop below
		})
	}
	dups := make(map[string]bool)
	for _, epnt := range epnts {
		id := epnt.String()
		if _, found := dups[id]; found {
			continue
		}
		dups[id] = true
		epnt.Header = h.onlyAllowedHeaders(header)
		out = append(out, epnt)
	}
	return
}

func (h *THHandler) onlyAllowedHeaders(header http.Header) (out http.Header) {
	out = http.Header{}
	for k, vv := range header {
		switch strings.ToLower(k) {
		case "accept", "accept-language", "user-agent":
			for _, v := range vv {
				out.Add(k, v)
			}
		default:
			// ignore all the other headers
		}
	}
	return
}

// maybeQUICFollowUp checks whether we need to use Alt-Svc to check
// for QUIC. We query for HTTPSSvc but currently only Cloudflare
// implements this proposed standard. So, this function is
// where we take care of all the other servers implementing QUIC.
func (h *THHandler) maybeQUICFollowUp(ctx context.Context,
	m *THServerResponse, epnts ...*HTTPEndpoint) {
	altsvc := []string{}
	for _, epnt := range m.Endpoints {
		// Check whether we have a QUIC handshake. If so, then
		// HTTPSSvc worked and we can stop here.
		if epnt.QUICHandshake != nil {
			return
		}
		for _, rtrip := range epnt.HTTPRoundTrip {
			if v := rtrip.ResponseHeader.Get("alt-svc"); v != "" {
				altsvc = append(altsvc, v)
			}
		}
	}
	// syntax:
	//
	// Alt-Svc: clear
	// Alt-Svc: <protocol-id>=<alt-authority>; ma=<max-age>
	// Alt-Svc: <protocol-id>=<alt-authority>; ma=<max-age>; persist=1
	//
	// multiple entries may be separated by comma.
	//
	// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Alt-Svc
	for _, header := range altsvc {
		entries := strings.Split(header, ",")
		if len(entries) < 1 {
			continue
		}
		for _, entry := range entries {
			parts := strings.Split(entry, ";")
			if len(parts) < 1 {
				continue
			}
			if parts[0] == "h3=\":443\"" {
				h.doQUICFollowUp(ctx, m, epnts...)
				return
			}
		}
	}
}

// doQUICFollowUp runs when we know there's QUIC support via Alt-Svc.
func (h *THHandler) doQUICFollowUp(ctx context.Context,
	m *THServerResponse, epnts ...*HTTPEndpoint) {
	quicEpnts := []*HTTPEndpoint{}
	// do not mutate the existing list rather create a new one
	for _, epnt := range epnts {
		quicEpnts = append(quicEpnts, &HTTPEndpoint{
			Domain:  epnt.Domain,
			Network: NetworkQUIC,
			Address: epnt.Address,
			SNI:     epnt.SNI,
			ALPN:    []string{"h3"},
			URL:     epnt.URL,
			Header:  epnt.Header,
		})
	}
	mx := NewMeasurerWithDefaultSettings()
	jar := NewCookieJar()
	for me := range mx.HTTPEndpointGetParallel(ctx, jar, quicEpnts...) {
		m.Endpoints = append(m.Endpoints, h.newTHEndpointMeasurement(me))
	}
}

//
// TH server: marshalling of endpoint measurements
//

// newTHEndpointMeasurement takes in input an endpoint
// measurement performed by a measurer and emits in output
// the simplified THEndpointMeasurement equivalent.
func (h *THHandler) newTHEndpointMeasurement(in *Measurement) *THEndpointMeasurement {
	return &THEndpointMeasurement{
		Oddities:      in.Oddities,
		Connect:       h.newTHConnectEventList(in.Connect),
		TLSHandshake:  h.newTLSHandshakesList(in.TLSHandshake),
		QUICHandshake: h.newQUICHandshakeList(in.QUICHandshake),
		HTTPRoundTrip: h.newHTTPRoundTripList(in.HTTPRoundTrip),
	}
}

func (h *THHandler) newTHConnectEventList(in []*NetworkEvent) (out []*THConnectEvent) {
	for _, e := range in {
		out = append(out, &THConnectEvent{
			Network:    e.Network,
			RemoteAddr: e.RemoteAddr,
			Error:      h.errorToFailure(e.Error),
			Oddity:     e.Oddity,
		})
	}
	return
}

func (h *THHandler) newTLSHandshakesList(in []*TLSHandshakeEvent) (out []*THHandshakeEvent) {
	for _, e := range in {
		out = append(out, &THHandshakeEvent{
			Network:         e.Network,
			RemoteAddr:      e.RemoteAddr,
			SNI:             e.SNI,
			ALPN:            e.ALPN,
			Error:           h.errorToFailure(e.Error),
			Oddity:          e.Oddity,
			TLSVersion:      e.TLSVersion,
			CipherSuite:     e.CipherSuite,
			NegotiatedProto: e.NegotiatedProto,
		})
	}
	return
}

func (h *THHandler) newQUICHandshakeList(in []*QUICHandshakeEvent) (out []*THHandshakeEvent) {
	for _, e := range in {
		out = append(out, &THHandshakeEvent{
			Network:         e.Network,
			RemoteAddr:      e.RemoteAddr,
			SNI:             e.SNI,
			ALPN:            e.ALPN,
			Error:           h.errorToFailure(e.Error),
			Oddity:          e.Oddity,
			TLSVersion:      e.TLSVersion,
			CipherSuite:     e.CipherSuite,
			NegotiatedProto: e.NegotiatedProto,
		})
	}
	return
}

func (h *THHandler) newHTTPRoundTripList(in []*HTTPRoundTripEvent) (out []*THHTTPRoundTripEvent) {
	for _, e := range in {
		out = append(out, &THHTTPRoundTripEvent{
			RequestMethod:            e.RequestMethod,
			RequestURL:               e.RequestURL.String(),
			RequestHeader:            e.RequestHeader,
			Error:                    h.errorToFailure(e.Error),
			Oddity:                   e.Oddity,
			ResponseStatus:           int64(e.ResponseStatus),
			ResponseHeader:           e.ResponseHeader,
			ResponseBodySnapshotSize: int64(len(e.ResponseBodySnapshot)),
			MaxBodySnapshotSize:      e.MaxBodySnapshotSize,
		})
	}
	return
}

//
// TH server: DNS
//

// thResolverURL is the DNS resolver URL used by the TH. We use an
// encrypted resolver to reduce the risk that there is DNS-over-UDP
// censorship in the place where we deploy the TH.
const thResolverURL = "https://dns.google/dns-query"

// thResolver is the DNS resolver used by the TH.
//
// Here we're using github.com/apex/log as the logger, which
// is fine because this is backend only code.
var thResolver = netxlite.WrapResolver(log.Log, dnsx.NewSerialResolver(
	dnsx.NewDNSOverHTTPS(http.DefaultClient, thResolverURL),
))

// dohQuery discovers endpoints for the URL's hostname using DoH.
//
// Arguments:
//
// - ctx is the context for deadline/cancellation/timeout
//
// - parsedURL is the parsed URL
//
// Returns:
//
// - a possibly empty list of HTTPEndpoints (this happens for
// example if the URL's hostname causes NXDOMAIN)
//
// - the THDNSMeasurement for the THServeResponse message
func (h *THHandler) dohQuery(ctx context.Context, URL *url.URL) (
	epnts []*HTTPEndpoint, meas *THDNSMeasurement) {
	db := NewDB(time.Now()) // timing is not sent back to client
	r := WrapResolver(0, OriginTH, db, thResolver)
	meas = &THDNSMeasurement{}
	op := newOperationLogger(log.Log,
		"dohQuery A/AAAA for %s with %s", URL.Hostname(), r.Address())
	_, err := r.LookupHost(ctx, URL.Hostname())
	op.Stop(err)
	meas.LookupHost = h.newTHLookupHostList(db)
	switch URL.Scheme {
	case "https":
		op := newOperationLogger(log.Log,
			"dohQuery HTTPSSvc for %s with %s", URL.Hostname(), r.Address())
		_, err = r.LookupHTTPSSvcWithoutRetry(ctx, URL.Hostname())
		op.Stop(err)
		meas.LookupHTTPSSvc = h.newTHLookupHTTPSSvcList(db)
	default:
		// nothing
	}
	epnts, _ = db.SelectAllHTTPEndpointsForURL(URL) // nil on failure
	return
}

func (h *THHandler) newTHLookupHostList(db *DB) (out []*THLookupHostEvent) {
	for _, entry := range db.SelectAllFromLookupHost() {
		out = append(out, &THLookupHostEvent{
			Network: entry.Network,
			Address: entry.Address,
			Domain:  entry.Domain,
			Error:   h.errorToFailure(entry.Error),
			Oddity:  entry.Oddity,
			Addrs:   entry.Addrs,
		})
	}
	return
}

func (h *THHandler) newTHLookupHTTPSSvcList(db *DB) (out []*THLookupHTTPSSvcEvent) {
	for _, entry := range db.SelectAllFromLookupHTTPSSvc() {
		out = append(out, &THLookupHTTPSSvcEvent{
			Network: entry.Network,
			Address: entry.Address,
			Domain:  entry.Domain,
			Error:   h.errorToFailure(entry.Error),
			Oddity:  entry.Oddity,
			IPv4:    entry.IPv4,
			IPv6:    entry.IPv6,
			ALPN:    entry.ALPN,
		})
	}
	return
}

//
// TH server: utility functions
//

// errorToFailure converts an error type to a failure type (which
// is loosely defined as a pointer to a string).
//
// When the error is nil, the string pointer is nil. When the error is
// not nil, the pointer points to the err.Error() string.
//
// We cannot unmarshal Go errors from JSON. Therefore, we need to
// convert to this type when we're marshalling.
func (h *THHandler) errorToFailure(err error) (out *string) {
	if err != nil {
		s := err.Error()
		out = &s
	}
	return
}
