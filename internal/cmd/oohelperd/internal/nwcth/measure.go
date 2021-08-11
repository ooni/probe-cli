package nwcth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/nwebconnectivity"
)

// ControlResponse is the response from the control service.
type ControlResponse struct {
	URLMeasurements []*URLMeasurement `json:"urls"`
}

type (
	// ControlRequest is the request sent to the test helper
	ControlRequest = nwebconnectivity.ControlRequest

	// HTTPMeasurement contains the measurement for one URL, and one IP endpoint,
	// using HTTP over TCP.
	HTTPMeasurement = nwebconnectivity.ControlHTTPMeasurement

	// H3Measurement contains the measurement for one URL, and one IP endpoint,
	// using HTTP over QUIC (HTTP/3).
	H3Measurement = nwebconnectivity.ControlH3Measurement
)

// ErrNoValidIP means that the DNS step failed and the client did not provide IP endpoints for testing.
var ErrNoValidIP = errors.New("no valid IP address to measure")

// supportedQUICVersion are the H3 over QUIC versions we currently support
var supportedQUICVersions = map[string]bool{
	"h3":    true,
	"h3-29": true,
}

func Measure(ctx context.Context, creq *ControlRequest) (*ControlResponse, error) {
	var (
		URL *url.URL
		err error
	)
	if URL, err = InitialChecks(creq.HTTPRequest); err != nil {
		log.Fatalf("initial checks failed: %s", err.Error())
	}
	rts, err := Explore(URL)
	if err != nil {
		log.Fatalf("explore failed: %s", err.Error())
	}
	meas, err := Generate(rts)
	if err != nil {
		log.Fatalf("generate failed: %s", err.Error())
	}
	for _, m := range meas {
		fmt.Printf("# %s\n", m.URL)
		fmt.Printf("method: %s\n", m.RoundTrip.Request.Method)
		fmt.Printf("url: %s\n", m.RoundTrip.Request.URL.String())
		fmt.Printf("headers: %+v\n", m.RoundTrip.Request.Header)
		fmt.Printf("dns: %+v\n", m.DNS)
	}
	return &ControlResponse{URLMeasurements: meas}, nil
}

// var measureFactory = map[string]func(ctx context.Context, creq *ControlRequest, endpoint string, cookiejar http.CookieJar, wg *sync.WaitGroup) EndpointMeasurement{
// 	"http":  measureHTTP,
// 	"https": measureHTTP,
// 	"h3":    measureH3,
// 	"h3-29": measureH3,
// }

// func measureHTTP(
// 	ctx context.Context,
// 	creq *ControlRequest,
// 	endpoint string,
// 	cookiejar http.CookieJar,
// 	wg *sync.WaitGroup,
// ) EndpointMeasurement {
// 	URL, err := url.Parse(creq.HTTPRequest)
// 	runtimex.PanicOnError(err, "url.Parse failed")
// 	httpMeasurement := &HTTPMeasurement{Endpoint: endpoint, Protocol: URL.Scheme}

// 	var conn net.Conn
// 	conn, httpMeasurement.TCPConnect = TCPDo(ctx, &TCPConfig{
// 		Endpoint: endpoint,
// 	})
// 	if conn == nil {
// 		return httpMeasurement
// 	}
// 	defer conn.Close()
// 	var transport http.RoundTripper
// 	switch URL.Scheme {
// 	case "http":
// 		transport = nwebconnectivity.NewSingleTransport(conn, nil)
// 	case "https":
// 		var tlsconn *tls.Conn
// 		cfg := &tls.Config{
// 			ServerName: URL.Hostname(),
// 			NextProtos: []string{"h2", "http/1.1"},
// 		}
// 		tlsconn, httpMeasurement.TLSHandshake = TLSDo(ctx, &TLSConfig{
// 			Conn:     conn,
// 			Endpoint: endpoint,
// 			Cfg:      cfg,
// 		})
// 		if tlsconn == nil {
// 			return httpMeasurement
// 		}
// 		transport = nwebconnectivity.NewSingleTransport(tlsconn, cfg)
// 	}
// 	// perform the HTTP request: this provides us with the HTTP request result and info about HTTP redirection
// 	httpMeasurement.HTTPRequest = HTTPDo(ctx, &HTTPConfig{
// 		Jar:       cookiejar,
// 		Headers:   creq.HTTPRequestHeaders,
// 		Transport: transport,
// 		URL:       URL,
// 	})
// 	return httpMeasurement
// }

// func measureH3(
// 	ctx context.Context,
// 	creq *ControlRequest,
// 	endpoint string,
// 	cookiejar http.CookieJar,
// 	wg *sync.WaitGroup,
// ) EndpointMeasurement {
// 	URL, err := url.Parse(creq.HTTPRequest)
// 	runtimex.PanicOnError(err, "url.Parse failed")
// 	h3Measurement := &H3Measurement{Endpoint: endpoint, Protocol: URL.Scheme}

// 	var sess quic.EarlySession
// 	tlscfg := &tls.Config{
// 		ServerName: URL.Hostname(),
// 		NextProtos: []string{URL.Scheme},
// 	}
// 	qcfg := &quic.Config{}
// 	sess, h3Measurement.QUICHandshake = QUICDo(ctx, &QUICConfig{
// 		Endpoint:  endpoint,
// 		QConfig:   qcfg,
// 		TLSConfig: tlscfg,
// 	})
// 	if sess == nil {
// 		return h3Measurement
// 	}
// 	transport := nwebconnectivity.NewSingleH3Transport(sess, tlscfg, qcfg)
// 	h3Measurement.HTTPRequest = HTTPDo(ctx, &HTTPConfig{
// 		Jar:       cookiejar,
// 		Headers:   creq.HTTPRequestHeaders,
// 		Transport: transport,
// 		URL:       URL,
// 	})
// 	return h3Measurement
// }

// // mergeEndpoints creates a (duplicate-free) union set of the IP endpoints provided by the client,
// // and the IP endpoints resulting from the testhelper's DNS step
// func mergeEndpoints(endpoints []string, clientEndpoints []string) (out []string) {
// 	unique := make(map[string]bool, len(endpoints)+len(clientEndpoints))
// 	for _, a := range endpoints {
// 		unique[a] = true
// 	}
// 	for _, a := range clientEndpoints {
// 		unique[a] = true
// 	}
// 	for key := range unique {
// 		out = append(out, key)
// 	}
// 	return out
// }

// // getEndpoints connects IP addresses with the port associated with the URL scheme
// func getEndpoints(addrs []string, URL *url.URL) []string {
// 	out := []string{}
// 	_, h3ok := supportedQUICVersions[URL.Scheme]
// 	if URL.Scheme != "http" && URL.Scheme != "https" && !h3ok {
// 		panic("passed an unexpected scheme")
// 	}
// 	p := URL.Port()
// 	for _, a := range addrs {
// 		var port string
// 		switch {
// 		case p != "":
// 			// explicit port
// 			port = p
// 		case URL.Scheme == "http":
// 			port = "80"
// 		case URL.Scheme == "https":
// 			port = "443"
// 		case h3ok:
// 			port = "443"
// 		}
// 		endpoint := net.JoinHostPort(a, port)
// 		out = append(out, endpoint)
// 	}
// 	return out
// }

// // getNextURLs inspects the HTTP response headers and returns a slice of URLs to be measured next
// // Follow-up measurements can be either HTTP Redirect requests,
// // or HTTP/3 requests in case the host supports HTTP/3.
// func getNextURLs(urlM *URLMeasurement, URL *url.URL) (locations []*url.URL) {
// 	for _, u := range urlM.Endpoints {
// 		httpMeasurement := u.GetHTTPRequestMeasurement()
// 		if httpMeasurement == nil {
// 			continue
// 		}
// 		redirection := getRedirectionURL(httpMeasurement)
// 		if redirection != nil {
// 			locations = append(locations, redirection)
// 		}
// 		h3location := getH3URL(httpMeasurement, URL)
// 		if h3location != nil {
// 			locations = append(locations, h3location)
// 		}
// 	}
// 	return locations
// }

// func getRedirectionURL(httpMeasurement *HTTPRequestMeasurement) *url.URL {
// 	switch httpMeasurement.StatusCode {
// 	case 301, 302, 303, 307, 308:
// 	default:
// 		return nil
// 	}
// 	loc := httpMeasurement.Headers.Get("Location")
// 	if loc == "" {
// 		return nil
// 	}
// 	URL, err := url.Parse(loc)
// 	runtimex.PanicOnError(err, "url.Parse failed")
// 	URL.Scheme = realSchemes[URL.Scheme]
// 	return URL
// }

// func getH3URL(httpMeasurement *HTTPRequestMeasurement, URL *url.URL) *url.URL {
// 	h3Svc := parseAltSvc(httpMeasurement, URL)
// 	if h3Svc == nil {
// 		return nil
// 	}
// 	quicURL, err := url.Parse(URL.String())
// 	runtimex.PanicOnError(err, "url.Parse failed")
// 	quicURL.Scheme = h3Svc.proto
// 	quicURL.Host = h3Svc.authority
// 	return quicURL
// }
