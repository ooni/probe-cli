package nwcth

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/nwebconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

type (
	// CtrlRequest is the request sent to the test helper
	CtrlRequest = nwebconnectivity.ControlRequest

	// CtrlResponse is the response from the test helper
	CtrlResponse = nwebconnectivity.ControlResponse

	CtrlURLMeasurement = nwebconnectivity.ControlURL

	CtrlEndpointMeasurement = nwebconnectivity.ControlEndpoint

	CtrlHTTPMeasurement = nwebconnectivity.ControlHTTP

	CtrlH3Measurement = nwebconnectivity.ControlH3

	CtrlTLSMeasurement = nwebconnectivity.ControlTLSHandshake

	CtrlHTTPRequest = nwebconnectivity.ControlHTTPRequest
)

// MeasureConfig contains configuration for Measure.
type MeasureConfig struct {
	Dialer            netx.Dialer
	MaxAcceptableBody int64
	QuicDialer        netx.QUICDialer
	Resolver          netx.Resolver
}

// NextLocationInfo contains the redirected location as well as the request object which forwards most headers of the initial request.
// This forwarded request is generated by the http.Client and
type NextLocationInfo struct {
	jar             *cookiejar.Jar `json:"-"`
	location        string         `json:"-"`
	httpRedirectReq *http.Request  `json:"-"`
}

type MeasureURLResult struct {
	CtrlURLMeasurement *CtrlURLMeasurement `json:"-"`
	redirectedReqs     []*CtrlRequest      `json:"-"`
	h3Reqs             []*CtrlRequest      `json:"-"`
}

type MeasureEndpointResult struct {
	CtrlEndpoint CtrlEndpointMeasurement
	httpRedirect *NextLocationInfo
	h3Location   string
}

func Measure(ctx context.Context, config MeasureConfig, creq *CtrlRequest) (*CtrlResponse, error) {
	var cresp = &CtrlResponse{URLMeasurements: []*CtrlURLMeasurement{}}

	redirected := make(map[string]bool, 100)

	urlM, err := MeasureURL(ctx, config, creq, cresp, redirected)
	if err != nil {
		return nil, err
	}
	cresp.URLMeasurements = append(cresp.URLMeasurements, urlM.CtrlURLMeasurement)

	n := 0
	nextRequests := append(urlM.redirectedReqs, urlM.h3Reqs...)
	for len(nextRequests) > n {
		req := nextRequests[n]
		n += 1
		if _, ok := redirected[req.HTTPRequest]; ok {
			continue
		}
		redirected[req.HTTPRequest] = true
		urlM, err := MeasureURL(ctx, config, req, cresp, redirected)
		if err != nil {
			return nil, err
		}
		cresp.URLMeasurements = append(cresp.URLMeasurements, urlM.CtrlURLMeasurement)
		nextRequests = append(nextRequests, urlM.redirectedReqs...)
	}

	return cresp, nil
}

// Measure performs the measurement described by the request and
// returns the corresponding response or an error.
func MeasureURL(ctx context.Context, config MeasureConfig, creq *CtrlRequest, cresp *CtrlResponse, redirected map[string]bool) (*MeasureURLResult, error) {
	// parse input for correctness
	URL, err := url.Parse(creq.HTTPRequest)
	if err != nil {
		return nil, err
	}

	// create URLMeasurement struct
	urlMeasurement := &CtrlURLMeasurement{
		URL:       URL.String(),
		DNS:       nil,
		Endpoints: []CtrlEndpointMeasurement{},
	}

	// dns: start
	dns := DNSDo(ctx, &DNSConfig{
		Domain:   URL.Hostname(),
		Resolver: config.Resolver,
	})

	urlMeasurement.DNS = &dns

	enpnts := getEndpoints(dns.Addrs, URL)
	addrs := mergeEndpoints(enpnts, creq.TCPConnect)

	if len(addrs) == 0 {
		return nil, errors.New("no valid IP address to measure")
	}

	wg := new(sync.WaitGroup)
	out := make(chan *MeasureEndpointResult, len(enpnts))
	for _, endpoint := range enpnts {
		wg.Add(1)
		go MeasureEndpoint(ctx, config, creq, URL, endpoint, wg, out)
	}
	wg.Wait()
	close(out)

	h3Reqs := []*CtrlRequest{}
	redirectedReqs := []*CtrlRequest{}
	for m := range out {
		urlMeasurement.Endpoints = append(urlMeasurement.Endpoints, m.CtrlEndpoint)
		if m.httpRedirect != nil {
			if len(redirected) == 20 {
				// stop after 20 redirects
				continue
			}
			req := &CtrlRequest{HTTPCookieJar: m.httpRedirect.jar, HTTPRequest: m.httpRedirect.location, HTTPRequestHeaders: m.httpRedirect.httpRedirectReq.Header}
			redirectedReqs = append(redirectedReqs, req)
		}
		if m.h3Location != "" {
			req := &CtrlRequest{HTTPRequest: m.h3Location}
			h3Reqs = append(h3Reqs, req)
		}
	}
	return &MeasureURLResult{CtrlURLMeasurement: urlMeasurement, h3Reqs: h3Reqs, redirectedReqs: redirectedReqs}, nil
}

func MeasureEndpoint(ctx context.Context, config MeasureConfig, creq *CtrlRequest, URL *url.URL, endpoint string, wg *sync.WaitGroup, out chan *MeasureEndpointResult) {
	defer wg.Done()
	endpointResult := &MeasureEndpointResult{}
	measureFactory[URL.Scheme](ctx, config, creq, endpoint, wg, endpointResult)
	out <- endpointResult
}

var measureFactory = map[string]func(ctx context.Context, config MeasureConfig, creq *CtrlRequest, endpoint string, wg *sync.WaitGroup, result *MeasureEndpointResult){
	"http":  measureHTTP,
	"https": measureHTTP,
	"h3":    measureH3,
	"h3-29": measureH3,
}

func measureHTTP(
	ctx context.Context,
	config MeasureConfig,
	creq *CtrlRequest,
	endpoint string,
	wg *sync.WaitGroup,
	result *MeasureEndpointResult,
) {
	URL, _ := url.Parse(creq.HTTPRequest)
	httpMeasurement := CtrlHTTPMeasurement{Endpoint: endpoint, Protocol: URL.Scheme}
	var conn net.Conn
	conn, httpMeasurement.TCPConnect = TCPDo(ctx, &TCPConfig{
		Dialer:   config.Dialer,
		Endpoint: endpoint,
	})
	if conn == nil {
		return
	}
	defer conn.Close()
	var transport http.RoundTripper
	switch URL.Scheme {
	case "http":
		transport = nwebconnectivity.GetSingleTransport(nil, conn, nil)
	case "https":
		var tlsconn *tls.Conn
		cfg := &tls.Config{ServerName: URL.Hostname()}
		tlsconn, httpMeasurement.TLSHandshake = TLSDo(ctx, &TLSConfig{
			Conn:     conn,
			Endpoint: endpoint,
			Cfg:      cfg,
		})
		if tlsconn == nil {
			return
		}
		state := tlsconn.ConnectionState()
		transport = nwebconnectivity.GetSingleTransport(&state, tlsconn, cfg)
	}
	// perform the HTTP request: this provides us with the HTTP request result and info about HTTP redirection
	httpMeasurement.HTTPRequest, result.httpRedirect = HTTPDo(ctx, &HTTPConfig{
		Jar:               creq.HTTPCookieJar,
		Headers:           creq.HTTPRequestHeaders,
		MaxAcceptableBody: config.MaxAcceptableBody,
		Transport:         transport,
		URL:               creq.HTTPRequest,
	})
	// find out of the host also supports h3 support, which is announced in the Alt-Svc Header
	h3Support := discoverH3Server(&httpMeasurement, URL)
	if h3Support != "" {
		quicURL, _ := url.Parse(URL.String())
		quicURL.Scheme = h3Support
		result.h3Location = quicURL.String()
	}
	result.CtrlEndpoint = &httpMeasurement
}

func measureH3(
	ctx context.Context,
	config MeasureConfig,
	creq *CtrlRequest,
	endpoint string,
	wg *sync.WaitGroup,
	result *MeasureEndpointResult,
) {
	URL, _ := url.Parse(creq.HTTPRequest)
	h3Measurement := CtrlH3Measurement{Endpoint: endpoint, Protocol: URL.Scheme}
	var sess quic.EarlySession
	tlscfg := &tls.Config{ServerName: URL.Hostname(), NextProtos: []string{URL.Scheme}}
	qcfg := &quic.Config{}
	sess, h3Measurement.QUICHandshake = QUICDo(ctx, &QUICConfig{
		Dialer:    config.QuicDialer,
		Endpoint:  endpoint,
		QConfig:   qcfg,
		TLSConfig: tlscfg,
	})
	if sess == nil {
		return
	}
	transport := nwebconnectivity.GetSingleH3Transport(sess, tlscfg, qcfg)
	h3Measurement.HTTPRequest, result.httpRedirect = HTTPDo(ctx, &HTTPConfig{
		Jar:               creq.HTTPCookieJar,
		Headers:           creq.HTTPRequestHeaders,
		MaxAcceptableBody: config.MaxAcceptableBody,
		Transport:         transport,
		URL:               "https://" + URL.Hostname(),
	})
	sess.CloseWithError(0, "")
	result.CtrlEndpoint = &h3Measurement
}

func mergeEndpoints(addrs []string, clientAddrs []string) []string {
	appendIfUnique := func(slice []string, item string) []string {
		for _, i := range slice {
			if i == item {
				return slice
			}
		}
		return append(slice, item)
	}
	for _, c := range clientAddrs {
		addrs = appendIfUnique(addrs, c)
	}
	return addrs
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
