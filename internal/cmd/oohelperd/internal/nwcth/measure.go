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
	Client            *http.Client
	Dialer            netx.Dialer
	MaxAcceptableBody int64
	QuicDialer        netx.QUICDialer
	Resolver          netx.Resolver
}

func Measure(ctx context.Context, config MeasureConfig, creq *CtrlRequest) (*CtrlResponse, error) {
	var cresp = CtrlResponse{URLMeasurements: []*CtrlURLMeasurement{}}
	nexturlch := make(chan *NextLocationInfo, 20)
	nRedirects := 0

	urlM, err := MeasureURL(ctx, config, creq, nexturlch)
	if err != nil {
		return nil, err
	}
	cresp.URLMeasurements = append(cresp.URLMeasurements, urlM)

	redirected := make(map[string]bool, 100)
	rdrctreqs := reduceRedirects(nexturlch, redirected, creq)

	n := 0
	for len(rdrctreqs) > n {
		next := rdrctreqs[n]
		n += 1
		headers := creq.HTTPRequestHeaders
		// this is an HTTP redirect
		if next.HTTPRedirectReq != nil {
			headers = next.HTTPRedirectReq.Header
			if nRedirects == 20 {
				// we stop after 20 redirects, as do Chrome and Firefox, TODO(kelmenhorst): how do we test this?
				// TODO(kelmenhorst): do we need another entry indicating the redirect failure here?
				break
			}
			nRedirects += 1
		}
		req := &CtrlRequest{HTTPRequest: next.Location.String(), TCPConnect: []string{}, HTTPRequestHeaders: headers}
		nexturlch = make(chan *NextLocationInfo, 20)
		urlM, err = MeasureURL(ctx, config, req, nexturlch)
		if err != nil {
			return nil, err
		}
		cresp.URLMeasurements = append(cresp.URLMeasurements, urlM)
		rdrctreqs = append(rdrctreqs, reduceRedirects(nexturlch, redirected, creq)...)
	}
	return &cresp, nil
}

func reduceRedirects(nexturlch chan *NextLocationInfo, redirected map[string]bool, creq *CtrlRequest) []*NextLocationInfo {
	out := []*NextLocationInfo{}
	for rdrct := range nexturlch {
		if _, ok := redirected[rdrct.Location.String()]; ok {
			continue
		}
		redirected[rdrct.Location.String()] = true
		out = append(out, rdrct)
	}
	return out
}

// Measure performs the measurement described by the request and
// returns the corresponding response or an error.
func MeasureURL(ctx context.Context, config MeasureConfig, creq *CtrlRequest, nexturlch chan *NextLocationInfo) (*CtrlURLMeasurement, error) {
	defer close(nexturlch)
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
	for _, endpoint := range enpnts {
		wg.Add(1)
		go func(endpoint string, wg *sync.WaitGroup) {
			defer wg.Done()
			endpointMeasurement := measureFactory[URL.Scheme](ctx, config, creq, endpoint, wg, nexturlch)
			urlMeasurement.Endpoints = append(urlMeasurement.Endpoints, endpointMeasurement)
			h3Support := discoverH3Server(endpointMeasurement, URL)
			if h3Support != "" {
				quicURL, _ := url.Parse(URL.String())
				quicURL.Scheme = h3Support
				nexturlch <- &NextLocationInfo{Location: quicURL}
			}
		}(endpoint, wg)
	}
	wg.Wait()
	return urlMeasurement, nil
}

var measureFactory = map[string]func(ctx context.Context, config MeasureConfig, creq *CtrlRequest, endpoint string, wg *sync.WaitGroup, nexturlch chan *NextLocationInfo) CtrlEndpointMeasurement{
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
	nexturlch chan *NextLocationInfo,
) CtrlEndpointMeasurement {
	URL, _ := url.Parse(creq.HTTPRequest)
	httpMeasurement := CtrlHTTPMeasurement{Endpoint: endpoint, Protocol: URL.Scheme}
	var conn net.Conn
	conn, httpMeasurement.TCPConnect = TCPDo(ctx, &TCPConfig{
		Dialer:   config.Dialer,
		Endpoint: endpoint,
	})
	if conn == nil {
		return &httpMeasurement
	}
	defer conn.Close()
	switch URL.Scheme {
	case "http":
		config.Client.Transport = nwebconnectivity.GetSingleTransport(nil, conn, nil)
	case "https":
		var tlsconn *tls.Conn
		cfg := &tls.Config{ServerName: URL.Hostname()}
		tlsconn, httpMeasurement.TLSHandshake = TLSDo(ctx, &TLSConfig{
			Conn:     conn,
			Endpoint: endpoint,
			Cfg:      cfg,
		})
		if tlsconn == nil {
			return &httpMeasurement
		}
		state := tlsconn.ConnectionState()
		config.Client.Transport = nwebconnectivity.GetSingleTransport(&state, tlsconn, cfg)
	}
	httpMeasurement.HTTPRequest = HTTPDo(ctx, &HTTPConfig{
		Client:            config.Client,
		Headers:           creq.HTTPRequestHeaders,
		MaxAcceptableBody: config.MaxAcceptableBody,
		URL:               creq.HTTPRequest,
	}, nexturlch)
	return &httpMeasurement
}

func measureH3(
	ctx context.Context,
	config MeasureConfig,
	creq *CtrlRequest,
	endpoint string,
	wg *sync.WaitGroup,
	nexturlch chan *NextLocationInfo,
) CtrlEndpointMeasurement {
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
		return &h3Measurement
	}
	transport := nwebconnectivity.GetSingleH3Transport(sess, tlscfg, qcfg)
	config.Client.Transport = transport
	h3Measurement.HTTPRequest = HTTPDo(ctx, &HTTPConfig{
		Client:            config.Client,
		Headers:           creq.HTTPRequestHeaders,
		MaxAcceptableBody: config.MaxAcceptableBody,
		URL:               "https://" + URL.Hostname(),
	}, nexturlch)
	sess.CloseWithError(0, "")
	return &h3Measurement
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
