package websteps

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

const (
	testName    = "web_steps"
	testVersion = "0.1.0"
)

// Config contains the experiment config.
type Config struct{}

// TestKeys contains webconnectivity test keys.
type TestKeys struct {
	Agent           string `json:"agent"`
	ClientResolver  string `json:"client_resolver"`
	URLMeasurements []*URLMeasurement
}

// Measurer performs the measurement.
type Measurer struct {
	Config Config
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return Measurer{Config: config}
}

// ExperimentName implements ExperimentMeasurer.ExperExperimentName.
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperExperimentVersion.
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

// SupportedQUICVersions are the H3 over QUIC versions we currently support
var SupportedQUICVersions = map[string]bool{
	"h3":    true,
	"h3-29": true,
}

var (
	// ErrNoAvailableTestHelpers is emitted when there are no available test helpers.
	ErrNoAvailableTestHelpers = errors.New("no available helpers")

	// ErrNoInput indicates that no input was provided
	ErrNoInput = errors.New("no input provided")

	// ErrInputIsNotAnURL indicates that the input is not an URL.
	ErrInputIsNotAnURL = errors.New("input is not an URL")

	// ErrUnsupportedInput indicates that the input URL scheme is unsupported.
	ErrUnsupportedInput = errors.New("unsupported input scheme")
)

// Run implements ExperimentMeasurer.Run.
func (m Measurer) Run(
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	tk := new(TestKeys)
	measurement.TestKeys = tk
	tk.Agent = "redirect"
	tk.ClientResolver = sess.ResolverIP()

	// 1. Parse and verify URL
	URL, err := url.Parse(string(measurement.Input))
	if err != nil {
		return ErrInputIsNotAnURL
	}
	if URL.Scheme != "http" && URL.Scheme != "https" {
		return ErrUnsupportedInput
	}
	// 2. Perform the initial DNS lookup step
	addrs, err := DNSDo(ctx, DNSConfig{Domain: URL.Hostname()})
	endpoints := makeEndpoints(addrs, URL)
	// 3. Find the testhelper
	testhelpers, _ := sess.GetTestHelpersByName("web-connectivity")
	var testhelper *model.Service
	for _, th := range testhelpers {
		if th.Type == "https" {
			testhelper = &th
			break
		}
	}
	if testhelper == nil {
		return ErrNoAvailableTestHelpers
	}
	measurement.TestHelpers = map[string]interface{}{
		"backend": testhelper,
	}
	// 4. Query the testhelper
	resp, err := Control(ctx, sess, testhelper.Address, CtrlRequest{
		HTTPRequest: URL.String(),
		HTTPRequestHeaders: map[string][]string{
			"Accept":          {httpheader.Accept()},
			"Accept-Language": {httpheader.AcceptLanguage()},
			"User-Agent":      {httpheader.UserAgent()},
		},
		Addrs: endpoints,
	})
	if err != nil || resp.URLMeasurements == nil {
		return errors.New("no control response")
	}
	// 5. Go over the Control URL measurements and reproduce them without following redirects, one by one.
	for _, controlURLMeasurement := range resp.URLMeasurements {
		urlMeasurement := &URLMeasurement{
			URL:       controlURLMeasurement.URL,
			Endpoints: []*EndpointMeasurement{},
		}
		URL, err = url.Parse(controlURLMeasurement.URL)
		runtimex.PanicOnError(err, "url.Parse failed")
		// DNS step
		addrs, err = DNSDo(ctx, DNSConfig{Domain: URL.Hostname()})
		urlMeasurement.DNS = &DNSMeasurement{
			Domain:  URL.Hostname(),
			Addrs:   addrs,
			Failure: archival.NewFailure(err),
		}
		if controlURLMeasurement.Endpoints == nil {
			tk.URLMeasurements = append(tk.URLMeasurements, urlMeasurement)
			continue
		}
		// the testhelper tells us which endpoints to measure
		for _, controlEndpoint := range controlURLMeasurement.Endpoints {
			rt := controlEndpoint.HTTPRoundTripMeasurement
			if rt == nil || rt.Request == nil {
				continue
			}
			var endpointMeasurement *EndpointMeasurement
			proto := controlEndpoint.Protocol
			_, h3 := SupportedQUICVersions[proto]
			switch {
			case h3:
				endpointMeasurement = m.measureEndpointH3(ctx, URL, controlEndpoint.Endpoint, rt.Request.Headers, proto)
			case proto == "http":
				endpointMeasurement = m.measureEndpointHTTP(ctx, URL, controlEndpoint.Endpoint, rt.Request.Headers)
			case proto == "https":
				endpointMeasurement = m.measureEndpointHTTPS(ctx, URL, controlEndpoint.Endpoint, rt.Request.Headers)
			default:
				panic("should not happen")
			}
			urlMeasurement.Endpoints = append(urlMeasurement.Endpoints, endpointMeasurement)
		}
		tk.URLMeasurements = append(tk.URLMeasurements, urlMeasurement)
	}
	return nil
}

func (m *Measurer) measureEndpointHTTP(ctx context.Context, URL *url.URL, endpoint string, headers http.Header) *EndpointMeasurement {
	endpointMeasurement := &EndpointMeasurement{
		Endpoint: endpoint,
		Protocol: "http",
	}
	// TCP connect step
	conn, err := TCPDo(ctx, TCPConfig{Endpoint: endpoint})
	endpointMeasurement.TCPConnectMeasurement = &TCPConnectMeasurement{
		Failure: archival.NewFailure(err),
	}
	if err != nil {
		return endpointMeasurement
	}
	defer conn.Close()

	// HTTP roundtrip step
	request := NewRequest(ctx, URL, headers)
	endpointMeasurement.HTTPRoundTripMeasurement = &HTTPRoundTripMeasurement{
		Request: &HTTPRequestMeasurement{
			Headers: request.Header,
			Method:  "GET",
			URL:     URL.String(),
		},
	}
	transport := NewSingleTransport(conn)
	resp, body, err := HTTPDo(request, transport)
	if err != nil {
		// failed Response
		endpointMeasurement.HTTPRoundTripMeasurement.Response = &HTTPResponseMeasurement{
			Failure: archival.NewFailure(err),
		}
		return endpointMeasurement
	}
	// successful Response
	endpointMeasurement.HTTPRoundTripMeasurement.Response = &HTTPResponseMeasurement{
		BodyLength: int64(len(body)),
		Failure:    nil,
		Headers:    resp.Header,
		StatusCode: int64(resp.StatusCode),
	}
	return endpointMeasurement
}

func (m *Measurer) measureEndpointHTTPS(ctx context.Context, URL *url.URL, endpoint string, headers http.Header) *EndpointMeasurement {
	endpointMeasurement := &EndpointMeasurement{
		Endpoint: endpoint,
		Protocol: "https",
	}
	// TCP connect step
	conn, err := TCPDo(ctx, TCPConfig{Endpoint: endpoint})
	endpointMeasurement.TCPConnectMeasurement = &TCPConnectMeasurement{
		Failure: archival.NewFailure(err),
	}
	if err != nil {
		return endpointMeasurement
	}
	defer conn.Close()

	// TLS handshake step
	tlsconn, err := TLSDo(conn, URL.Hostname())
	endpointMeasurement.TLSHandshakeMeasurement = &TLSHandshakeMeasurement{
		Failure: archival.NewFailure(err),
	}
	if err != nil {
		return endpointMeasurement
	}
	defer tlsconn.Close()

	// HTTP roundtrip step
	request := NewRequest(ctx, URL, headers)
	endpointMeasurement.HTTPRoundTripMeasurement = &HTTPRoundTripMeasurement{
		Request: &HTTPRequestMeasurement{
			Headers: request.Header,
			Method:  "GET",
			URL:     URL.String(),
		},
	}
	transport := NewSingleTransport(tlsconn)
	resp, body, err := HTTPDo(request, transport)
	if err != nil {
		// failed Response
		endpointMeasurement.HTTPRoundTripMeasurement.Response = &HTTPResponseMeasurement{
			Failure: archival.NewFailure(err),
		}
		return endpointMeasurement
	}
	// successful Response
	endpointMeasurement.HTTPRoundTripMeasurement.Response = &HTTPResponseMeasurement{
		BodyLength: int64(len(body)),
		Failure:    nil,
		Headers:    resp.Header,
		StatusCode: int64(resp.StatusCode),
	}
	return endpointMeasurement
}

func (m *Measurer) measureEndpointH3(ctx context.Context, URL *url.URL, endpoint string, headers http.Header, proto string) *EndpointMeasurement {
	endpointMeasurement := &EndpointMeasurement{
		Endpoint: endpoint,
		Protocol: proto,
	}
	tlsConf := &tls.Config{
		ServerName: URL.Hostname(),
		NextProtos: []string{proto},
	}
	// QUIC handshake step
	sess, err := QUICDo(ctx, QUICConfig{
		Endpoint: endpoint,
		TLSConf:  tlsConf,
	})
	endpointMeasurement.QUICHandshakeMeasurement = &TLSHandshakeMeasurement{
		Failure: archival.NewFailure(err),
	}
	if err != nil {
		return endpointMeasurement
	}
	// HTTP roundtrip step
	request := NewRequest(ctx, URL, headers)
	endpointMeasurement.HTTPRoundTripMeasurement = &HTTPRoundTripMeasurement{
		Request: &HTTPRequestMeasurement{
			Headers: request.Header,
			Method:  "GET",
			URL:     URL.String(),
		},
	}
	transport := NewSingleH3Transport(sess, tlsConf, &quic.Config{})
	resp, body, err := HTTPDo(request, transport)
	if err != nil {
		// failed Response
		endpointMeasurement.HTTPRoundTripMeasurement.Response = &HTTPResponseMeasurement{
			Failure: archival.NewFailure(err),
		}
		return endpointMeasurement
	}
	// successful Response
	endpointMeasurement.HTTPRoundTripMeasurement.Response = &HTTPResponseMeasurement{
		BodyLength: int64(len(body)),
		Failure:    nil,
		Headers:    resp.Header,
		StatusCode: int64(resp.StatusCode),
	}
	return endpointMeasurement

}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	Accessible bool   `json:"accessible"`
	Blocking   string `json:"blocking"`
	IsAnomaly  bool   `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{}
	return sk, nil
}

func makeEndpoints(addrs []string, URL *url.URL) []string {
	endpoints := []string{}
	if addrs == nil {
		return endpoints
	}
	for _, addr := range addrs {
		var port string
		explicitPort := URL.Port()
		scheme := URL.Scheme
		switch {
		case explicitPort != "":
			port = explicitPort
		case scheme == "http":
			port = "80"
		case scheme == "https":
			port = "443"
		default:
			panic("should not happen")
		}
		endpoints = append(endpoints, net.JoinHostPort(addr, port))
	}
	return endpoints
}
