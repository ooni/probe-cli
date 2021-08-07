package nwcth

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/nwebconnectivity"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type (
	// CtrlRequest is the request sent to the test helper
	CtrlRequest = nwebconnectivity.ControlRequest

	// CtrlResponse is the response from the test helper
	CtrlResponse = nwebconnectivity.ControlResponse

	// URLMeasurement contains the measurement for one URL
	URLMeasurement = nwebconnectivity.ControlURLMeasurement

	// EndpointMeasurement contains the measurement for one URL, and one IP endpoint.
	// It is either an HTTPMeasurement or an H3Measurement.
	EndpointMeasurement = nwebconnectivity.ControlEndpointMeasurement

	// HTTPMeasurement contains the measurement for one URL, and one IP endpoint,
	// using HTTP over TCP.
	HTTPMeasurement = nwebconnectivity.ControlHTTPMeasurement

	// H3Measurement contains the measurement for one URL, and one IP endpoint,
	// using HTTP over QUIC (HTTP/3).
	H3Measurement = nwebconnectivity.ControlH3Measurement

	// TLSHandshakeMeasurement contains the measurement of a single TLS handshake operation.
	// This can also be a QUIC handshake (which includes a TLS 1.3 handshake)
	TLSHandshakeMeasurement = nwebconnectivity.ControlTLSHandshakeMeasurement

	// HTTPRequestMeasurement contains the measurement of a single HTTP roundtrip operation.
	// The underlying transport protocol could be TCP (HTTP(S)) or QUIC (HTTP/3).
	HTTPRequestMeasurement = nwebconnectivity.ControlHTTPRequestMeasurement
)

// ErrNoValidIP means that the DNS step failed and the client did not provide IP endpoints for testing.
var ErrNoValidIP = errors.New("no valid IP address to measure")

// NextLocationInfo contains the redirected location,
// and the http cookiejar used for the redirect chain.
type NextLocationInfo struct {
	jar      http.CookieJar `json:"-"`
	location string         `json:"-"`
}

type MeasureURLResult struct {
	URLMeasurement *URLMeasurement `json:"-"`
	redirectedReqs []*CtrlRequest  `json:"-"`
	h3Reqs         []*CtrlRequest  `json:"-"`
}

type MeasureEndpointResult struct {
	CtrlEndpoint EndpointMeasurement
	httpRedirect *NextLocationInfo
	h3Location   string
}

func Measure(ctx context.Context, creq *CtrlRequest) (*CtrlResponse, error) {
	var cresp = &CtrlResponse{URLMeasurements: []*URLMeasurement{}}

	redirected := make(map[string]bool, 100)

	urlM, err := MeasureURL(ctx, creq, cresp)
	if err != nil {
		return nil, err
	}
	cresp.URLMeasurements = append(cresp.URLMeasurements, urlM.URLMeasurement)

	n := 0
	nextRequests := append(urlM.redirectedReqs, urlM.h3Reqs...)
	for len(nextRequests) > n {
		req := nextRequests[n]
		n += 1
		if len(redirected) == 20 {
			// stop after 20 redirects
			break
		}
		if _, ok := redirected[req.HTTPRequest]; ok {
			continue
		}
		redirected[req.HTTPRequest] = true
		urlM, err := MeasureURL(ctx, req, cresp)
		if err != nil {
			return nil, err
		}
		cresp.URLMeasurements = append(cresp.URLMeasurements, urlM.URLMeasurement)
		nextRequests = append(nextRequests, urlM.redirectedReqs...)
	}

	return cresp, nil
}

// Measure performs the measurement described by the request and
// returns the corresponding response or an error.
func MeasureURL(ctx context.Context, creq *CtrlRequest, cresp *CtrlResponse) (*MeasureURLResult, error) {
	// parse input for correctness
	URL, err := url.Parse(creq.HTTPRequest)
	if err != nil {
		return nil, err
	}

	// create URLMeasurement struct
	urlMeasurement := &URLMeasurement{
		URL:       URL.String(),
		DNS:       nil,
		Endpoints: []EndpointMeasurement{},
	}

	// dns: start
	dns := DNSDo(ctx, &DNSConfig{
		Domain: URL.Hostname(),
	})

	urlMeasurement.DNS = dns

	enpnts := getEndpoints(dns.Addrs, URL)
	addrs := mergeEndpoints(enpnts, creq.TCPConnect)

	if len(addrs) == 0 {
		return nil, ErrNoValidIP
	}

	wg := new(sync.WaitGroup)
	out := make(chan *MeasureEndpointResult, len(enpnts))
	for _, endpoint := range enpnts {
		wg.Add(1)
		go MeasureEndpoint(ctx, creq, URL, endpoint, wg, out)
	}
	wg.Wait()
	close(out)

	h3Reqs := []*CtrlRequest{}
	redirectedReqs := []*CtrlRequest{}
	for m := range out {
		urlMeasurement.Endpoints = append(urlMeasurement.Endpoints, m.CtrlEndpoint)
		if m.httpRedirect != nil {
			req := &CtrlRequest{HTTPCookieJar: m.httpRedirect.jar, HTTPRequest: m.httpRedirect.location}
			redirectedReqs = append(redirectedReqs, req)
		}
		if m.h3Location != "" {
			req := &CtrlRequest{HTTPRequest: m.h3Location}
			h3Reqs = append(h3Reqs, req)
		}
	}
	return &MeasureURLResult{URLMeasurement: urlMeasurement, h3Reqs: h3Reqs, redirectedReqs: redirectedReqs}, nil
}

func MeasureEndpoint(ctx context.Context, creq *CtrlRequest, URL *url.URL, endpoint string, wg *sync.WaitGroup, out chan *MeasureEndpointResult) {
	defer wg.Done()
	// endpointResult := &MeasureEndpointResult{}
	endpointResult := measureFactory[URL.Scheme](ctx, creq, endpoint, wg)
	out <- endpointResult
}

var measureFactory = map[string]func(ctx context.Context, creq *CtrlRequest, endpoint string, wg *sync.WaitGroup) *MeasureEndpointResult{
	"http":  measureHTTP,
	"https": measureHTTP,
	"h3":    measureH3,
	"h3-29": measureH3,
}

func measureHTTP(
	ctx context.Context,
	creq *CtrlRequest,
	endpoint string,
	wg *sync.WaitGroup,
) *MeasureEndpointResult {
	result := &MeasureEndpointResult{}
	URL, err := url.Parse(creq.HTTPRequest)
	runtimex.PanicOnError(err, "url.Parse failed")
	httpMeasurement := HTTPMeasurement{Endpoint: endpoint, Protocol: URL.Scheme}
	var conn net.Conn
	conn, httpMeasurement.TCPConnect = TCPDo(ctx, &TCPConfig{
		Endpoint: endpoint,
	})
	if conn == nil {
		return nil
	}
	defer conn.Close()
	var transport http.RoundTripper
	switch URL.Scheme {
	case "http":
		transport = nwebconnectivity.GetSingleTransport(conn, nil)
	case "https":
		var tlsconn *tls.Conn
		cfg := &tls.Config{ServerName: URL.Hostname()}
		tlsconn, httpMeasurement.TLSHandshake = TLSDo(ctx, &TLSConfig{
			Conn:     conn,
			Endpoint: endpoint,
			Cfg:      cfg,
		})
		if tlsconn == nil {
			return nil
		}
		transport = nwebconnectivity.GetSingleTransport(tlsconn, cfg)
	}
	// perform the HTTP request: this provides us with the HTTP request result and info about HTTP redirection
	httpMeasurement.HTTPRequest, result.httpRedirect = HTTPDo(ctx, &HTTPConfig{
		Jar:       creq.HTTPCookieJar,
		Headers:   creq.HTTPRequestHeaders,
		Transport: transport,
		URL:       URL,
	})
	// find out of the host also supports h3 support, which is announced in the Alt-Svc Header
	h3URL, err := getH3Location(httpMeasurement.HTTPRequest, URL)
	if err == nil {
		result.h3Location = h3URL.String()
	}
	result.CtrlEndpoint = &httpMeasurement
	return result
}

func measureH3(
	ctx context.Context,
	creq *CtrlRequest,
	endpoint string,
	wg *sync.WaitGroup,
) *MeasureEndpointResult {
	result := &MeasureEndpointResult{}
	URL, err := url.Parse(creq.HTTPRequest)
	runtimex.PanicOnError(err, "url.Parse failed")
	h3Measurement := H3Measurement{Endpoint: endpoint, Protocol: URL.Scheme}
	var sess quic.EarlySession
	tlscfg := &tls.Config{ServerName: URL.Hostname(), NextProtos: []string{URL.Scheme}}
	qcfg := &quic.Config{}
	sess, h3Measurement.QUICHandshake = QUICDo(ctx, &QUICConfig{
		Endpoint:  endpoint,
		QConfig:   qcfg,
		TLSConfig: tlscfg,
	})
	if sess == nil {
		return nil
	}
	transport := nwebconnectivity.GetSingleH3Transport(sess, tlscfg, qcfg)
	h3Measurement.HTTPRequest, result.httpRedirect = HTTPDo(ctx, &HTTPConfig{
		Jar:       creq.HTTPCookieJar,
		Headers:   creq.HTTPRequestHeaders,
		Transport: transport,
		URL:       URL,
	})
	sess.CloseWithError(0, "")
	result.CtrlEndpoint = &h3Measurement
	return result
}

func mergeEndpoints(addrs []string, clientAddrs []string) (out []string) {
	unique := make(map[string]bool, len(addrs)+len(clientAddrs))
	for _, a := range addrs {
		unique[a] = true
	}
	for _, a := range clientAddrs {
		unique[a] = true
	}
	for key := range unique {
		out = append(out, key)
	}
	return out
}

// getEndpoints connects IP addresses with the port associated with the URL scheme
func getEndpoints(addrs []string, URL *url.URL) []string {
	out := []string{}
	if URL.Scheme != "http" && URL.Scheme != "https" && URL.Scheme != "h3" && URL.Scheme != "h3-29" {
		panic("passed an unexpected scheme")
	}
	p := URL.Port()
	for _, a := range addrs {
		var port string
		switch true {
		case p != "":
			// explicit port
			port = p
		case URL.Scheme == "http":
			port = "80"
		case URL.Scheme == "https":
			port = "443"
		case URL.Scheme == "h3-29" || URL.Scheme == "h3":
			port = "443"
		}
		endpoint := net.JoinHostPort(a, port)
		out = append(out, endpoint)
	}
	return out
}
