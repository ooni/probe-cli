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

// supportedQUICVersion are the H3 over QUIC versions we currently support
var supportedQUICVersions = map[string]bool{
	"h3":    true,
	"h3-29": true,
}

func Measure(ctx context.Context, creq *CtrlRequest) (*CtrlResponse, error) {
	var cresp = &CtrlResponse{URLMeasurements: []*URLMeasurement{}}
	cookiejar, err := cookiejar.New(nil)
	runtimex.PanicOnError(err, "cookiejar.New failed")

	urlM, err := MeasureURL(ctx, creq, cresp, cookiejar)
	if err != nil {
		return nil, err
	}
	cresp.URLMeasurements = append(cresp.URLMeasurements, urlM)

	// this should not failed because otherwise we should have received an error in MeasureURL
	URL, err := url.Parse(creq.HTTPRequest)
	runtimex.PanicOnError(err, "url.Parse failed")

	// TODO(bassosimone,kelmenhorst): can this be further simplified?
	nextURLs := getNextURLs(urlM, URL)

	// the loop goes through the list of follow-up URL measurements and executes them
	// during the loop, the results of the follow-up measurements might add more follow-up requests to nextRequests
	// we stop, when there are no more follow-ups to perform
	visited := make(map[string]bool, 100)
	n := 0
	for len(nextURLs) > n {
		nextURL := nextURLs[n]
		n += 1
		// check if we have exceeded the maximum number of follow-up URLs
		if len(visited) == 20 {
			// stop after 20 follow-ups TODO(bassosimone,kelmenhorst): is that number reasonable?
			break
		}
		// check if we have already measured this particular URL
		if _, ok := visited[nextURL.String()]; ok {
			continue
		}
		visited[nextURL.String()] = true
		// perform follow-up URL measurement
		req := &CtrlRequest{HTTPRequest: nextURL.String()}
		urlM, err := MeasureURL(ctx, req, cresp, cookiejar)
		if err != nil {
			continue
		}
		cresp.URLMeasurements = append(cresp.URLMeasurements, urlM)
		// potentially add more triggered follow-up requests to the list
		nextURLs = append(nextURLs, getNextURLs(urlM, nextURL)...)
	}
	return cresp, nil
}

// getNextURLs inspects the HTTP response headers and returns a slice of URLs to be measured next
// Follow-up measurements can be either HTTP Redirect requests,
// or HTTP/3 requests in case the host supports HTTP/3.
func getNextURLs(urlM *URLMeasurement, URL *url.URL) (locations []*url.URL) {
	for _, u := range urlM.Endpoints {
		httpMeasurement := u.GetHTTPRequestMeasurement()
		if httpMeasurement == nil {
			continue
		}
		redirection := getRedirectionURL(httpMeasurement)
		if redirection != nil {
			locations = append(locations, redirection)
		}
		h3location := getH3URL(httpMeasurement, URL)
		if h3location != nil {
			locations = append(locations, h3location)
		}
	}
	return locations
}

func getRedirectionURL(httpMeasurement *HTTPRequestMeasurement) *url.URL {
	switch httpMeasurement.StatusCode {
	case 301, 302, 303, 307, 308:
	default:
		return nil
	}
	loc := httpMeasurement.Headers.Get("Location")
	if loc == "" {
		return nil
	}
	URL, err := url.Parse(loc)
	runtimex.PanicOnError(err, "url.Parse failed")
	URL.Scheme = realSchemes[URL.Scheme]
	return URL
}

func getH3URL(httpMeasurement *HTTPRequestMeasurement, URL *url.URL) *url.URL {
	h3Svc := parseAltSvc(httpMeasurement, URL)
	if h3Svc == nil {
		return nil
	}
	quicURL, err := url.Parse(URL.String())
	runtimex.PanicOnError(err, "url.Parse failed")
	quicURL.Scheme = h3Svc.proto
	quicURL.Host = h3Svc.authority
	return quicURL
}

// Measure performs the measurement described by the request and
// returns the corresponding response or an error.
func MeasureURL(ctx context.Context, creq *CtrlRequest, cresp *CtrlResponse, cookiejar http.CookieJar) (*URLMeasurement, error) {
	// create URLMeasurement struct
	urlMeasurement := &URLMeasurement{
		URL:       creq.HTTPRequest,
		DNS:       nil,
		Endpoints: []EndpointMeasurement{},
	}
	// parse input for correctness
	URL, err := url.Parse(creq.HTTPRequest)
	if err != nil {
		return urlMeasurement, err
	}

	// dns: start
	dns := DNSDo(ctx, &DNSConfig{
		Domain: URL.Hostname(),
	})

	urlMeasurement.DNS = dns

	enpnts := getEndpoints(dns.Addrs, URL)
	enpnts = mergeEndpoints(enpnts, creq.TCPConnect)

	if len(enpnts) == 0 {
		return nil, ErrNoValidIP
	}

	wg := new(sync.WaitGroup)
	out := make(chan EndpointMeasurement, len(enpnts))
	for _, endpoint := range enpnts {
		wg.Add(1)
		go MeasureEndpoint(ctx, creq, URL, endpoint, cookiejar, wg, out)
	}
	wg.Wait()
	close(out) // so iterating over it terminates (see below)
	for m := range out {
		urlMeasurement.Endpoints = append(urlMeasurement.Endpoints, m)
	}

	return urlMeasurement, nil
}

func MeasureEndpoint(ctx context.Context, creq *CtrlRequest, URL *url.URL, endpoint string, cookiejar http.CookieJar, wg *sync.WaitGroup, out chan EndpointMeasurement) {
	defer wg.Done()
	endpointResult := measureFactory[URL.Scheme](ctx, creq, endpoint, cookiejar, wg)
	out <- endpointResult
}

var measureFactory = map[string]func(ctx context.Context, creq *CtrlRequest, endpoint string, cookiejar http.CookieJar, wg *sync.WaitGroup) EndpointMeasurement{
	"http":  measureHTTP,
	"https": measureHTTP,
	"h3":    measureH3,
	"h3-29": measureH3,
}

func measureHTTP(
	ctx context.Context,
	creq *CtrlRequest,
	endpoint string,
	cookiejar http.CookieJar,
	wg *sync.WaitGroup,
) EndpointMeasurement {
	URL, err := url.Parse(creq.HTTPRequest)
	runtimex.PanicOnError(err, "url.Parse failed")
	httpMeasurement := &HTTPMeasurement{Endpoint: endpoint, Protocol: URL.Scheme}

	var conn net.Conn
	conn, httpMeasurement.TCPConnect = TCPDo(ctx, &TCPConfig{
		Endpoint: endpoint,
	})
	if conn == nil {
		return httpMeasurement
	}
	defer conn.Close()
	var transport http.RoundTripper
	switch URL.Scheme {
	case "http":
		transport = nwebconnectivity.NewSingleTransport(conn, nil)
	case "https":
		var tlsconn *tls.Conn
		cfg := &tls.Config{
			ServerName: URL.Hostname(),
			NextProtos: []string{"h2", "http/1.1"},
		}
		tlsconn, httpMeasurement.TLSHandshake = TLSDo(ctx, &TLSConfig{
			Conn:     conn,
			Endpoint: endpoint,
			Cfg:      cfg,
		})
		if tlsconn == nil {
			return httpMeasurement
		}
		transport = nwebconnectivity.NewSingleTransport(tlsconn, cfg)
	}
	// perform the HTTP request: this provides us with the HTTP request result and info about HTTP redirection
	httpMeasurement.HTTPRequest = HTTPDo(ctx, &HTTPConfig{
		Jar:       cookiejar,
		Headers:   creq.HTTPRequestHeaders,
		Transport: transport,
		URL:       URL,
	})
	return httpMeasurement
}

func measureH3(
	ctx context.Context,
	creq *CtrlRequest,
	endpoint string,
	cookiejar http.CookieJar,
	wg *sync.WaitGroup,
) EndpointMeasurement {
	URL, err := url.Parse(creq.HTTPRequest)
	runtimex.PanicOnError(err, "url.Parse failed")
	h3Measurement := &H3Measurement{Endpoint: endpoint, Protocol: URL.Scheme}

	var sess quic.EarlySession
	tlscfg := &tls.Config{
		ServerName: URL.Hostname(),
		NextProtos: []string{URL.Scheme},
	}
	qcfg := &quic.Config{}
	sess, h3Measurement.QUICHandshake = QUICDo(ctx, &QUICConfig{
		Endpoint:  endpoint,
		QConfig:   qcfg,
		TLSConfig: tlscfg,
	})
	if sess == nil {
		return h3Measurement
	}
	transport := nwebconnectivity.NewSingleH3Transport(sess, tlscfg, qcfg)
	h3Measurement.HTTPRequest = HTTPDo(ctx, &HTTPConfig{
		Jar:       cookiejar,
		Headers:   creq.HTTPRequestHeaders,
		Transport: transport,
		URL:       URL,
	})
	return h3Measurement
}

// mergeEndpoints creates a (duplicate-free) union set of the IP endpoints provided by the client,
// and the IP endpoints resulting from the testhelper's DNS step
func mergeEndpoints(endpoints []string, clientEndpoints []string) (out []string) {
	unique := make(map[string]bool, len(endpoints)+len(clientEndpoints))
	for _, a := range endpoints {
		unique[a] = true
	}
	for _, a := range clientEndpoints {
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
	_, h3ok := supportedQUICVersions[URL.Scheme]
	if URL.Scheme != "http" && URL.Scheme != "https" && !h3ok {
		panic("passed an unexpected scheme")
	}
	p := URL.Port()
	for _, a := range addrs {
		var port string
		switch {
		case p != "":
			// explicit port
			port = p
		case URL.Scheme == "http":
			port = "80"
		case URL.Scheme == "https":
			port = "443"
		case h3ok:
			port = "443"
		}
		endpoint := net.JoinHostPort(a, port)
		out = append(out, endpoint)
	}
	return out
}
