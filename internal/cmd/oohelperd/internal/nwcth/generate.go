package nwcth

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/nwebconnectivity"
)

// Generate is the third step of the algorithm. Given the
// observed round trips, we generate measurement targets and
// execute those measurements so the probe has a benchmark.

// URLMeasurement is a measurement of a given URL that
// includes connectivity measurement for each endpoint
// implied by the given URL.
type URLMeasurement struct {
	// URL is the URL we're using
	URL string

	// DNS contains the domain names resolved by the helper.
	DNS *DNSMeasurement

	// RoundTrip is the related round trip.
	RoundTrip *RoundTrip `json:"-"`

	// Endpoints contains endpoint measurements.
	Endpoints []EndpointMeasurement
}

// DNSMeasurement is a DNS measurement.
type DNSMeasurement struct {
	// Domain is the domain we wanted to resolve.
	Domain string

	// Addrs contains the resolved addresses.
	Addrs []string
}

type EndpointMeasurement interface {
	GetHTTPRoundtripMeasurement() *HTTPRoundtripMeasurement
}

// HTTPEndpointMeasurement is a measurement of a specific HTTP endpoint.
type HTTPEndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string

	// TCPConnectMeasurement is the related TCP connect measurement.
	TCPConnectMeasurement *TCPConnectMeasurement

	// TLSHandshakeMeasurement is the related TLS handshake measurement.
	TLSHandshakeMeasurement *TLSHandshakeMeasurement

	// HTTPRequestMeasurement is the related HTTP GET measurement.
	HTTPRoundtripMeasurement *HTTPRoundtripMeasurement
}

func (h *HTTPEndpointMeasurement) GetHTTPRoundtripMeasurement() *HTTPRoundtripMeasurement {
	return h.HTTPRoundtripMeasurement
}

// HTTPEndpointMeasurement is a measurement of a specific HTTP endpoint.
type H3EndpointMeasurement struct {
	// Endpoint is the endpoint we're measuring.
	Endpoint string

	// QUICHandshakeMeasurement is the related QUIC(TLS 1.3) handshake measurement.
	QUICHandshakeMeasurement *TLSHandshakeMeasurement

	// HTTPRequestMeasurement is the related HTTP GET measurement.
	HTTPRoundtripMeasurement *HTTPRoundtripMeasurement
}

func (h *H3EndpointMeasurement) GetHTTPRoundtripMeasurement() *HTTPRoundtripMeasurement {
	return h.HTTPRoundtripMeasurement
}

// Implementation note: OONI uses nil to indicate no error but here
// it's more convenient to just use an empty string.

// TCPConnectMeasurement is a TCP connect measurement.
type TCPConnectMeasurement struct {
	// Failure is the error that occurred.
	Failure *string
}

// TLSHandshakeMeasurement is a TLS handshake measurement.
type TLSHandshakeMeasurement struct {
	// Failure is the error that occurred.
	Failure *string
}

type HTTPRoundtripMeasurement struct {
	Request  *HTTPRequest
	Response *HTTPResponse
}

// HTTPRequestMeasurement is a HTTP request measurement.
type HTTPRequest struct {
	Headers http.Header
}

// HTTPRequestMeasurement is a HTTP request measurement.
type HTTPResponse struct {
	BodyLength int64       `json:"body_length"`
	Failure    *string     `json:"failure"`
	Headers    http.Header `json:"headers"`
	StatusCode int64       `json:"status_code"`
}

// Generate takes in input a list of round trips and outputs
// a list of connectivity measurements for each of them.
func Generate(ctx context.Context, rts []*RoundTrip) ([]*URLMeasurement, error) {
	var out []*URLMeasurement
	for _, rt := range rts {
		addrs, err := net.LookupHost(rt.Request.URL.Hostname())
		if err != nil {
			return nil, err
		}
		currentURL := &URLMeasurement{
			DNS: &DNSMeasurement{
				Domain: rt.Request.URL.Hostname(),
				Addrs:  addrs,
			},
			RoundTrip: rt,
			URL:       rt.Request.URL.String(),
		}
		out = append(out, currentURL)
		for _, addr := range addrs {
			// simplified algorithm to choose the port.
			var endpoint string
			switch rt.Request.URL.Scheme {
			case "http":
				endpoint = net.JoinHostPort(addr, "80")
			case "https":
				endpoint = net.JoinHostPort(addr, "443")
			default:
				panic("should not happen")
			}
			var currentEndpoint EndpointMeasurement
			_, h3 := supportedQUICVersions[rt.proto]
			switch {
			case h3:
				currentEndpoint = GenerateH3Endpoint(ctx, rt, endpoint)
			case rt.proto == "http":
				currentEndpoint = GenerateHTTPEndpoint(ctx, rt, endpoint)
			case rt.proto == "https":
				currentEndpoint = GenerateHTTPSEndpoint(ctx, rt, endpoint)
			default:
				// TODO(kelmenhorst): do we have to register this error somewhere in the result struct?
				continue
			}
			currentURL.Endpoints = append(currentURL.Endpoints, currentEndpoint)
		}
	}
	return out, nil
}

func GenerateHTTPEndpoint(ctx context.Context, rt *RoundTrip, endpoint string) EndpointMeasurement {
	currentEndpoint := &HTTPEndpointMeasurement{
		Endpoint: endpoint,
	}
	var tcpConn net.Conn
	tcpConn, currentEndpoint.TCPConnectMeasurement = TCPDo(ctx, endpoint)
	if tcpConn == nil {
		return currentEndpoint
	}
	defer tcpConn.Close() // suboptimal of course

	transport := nwebconnectivity.NewSingleTransport(tcpConn)
	currentEndpoint.HTTPRoundtripMeasurement = HTTPDo(rt.Request, transport)
	return currentEndpoint
}

func GenerateHTTPSEndpoint(ctx context.Context, rt *RoundTrip, endpoint string) EndpointMeasurement {
	currentEndpoint := &HTTPEndpointMeasurement{
		Endpoint: endpoint,
	}
	var tcpConn, tlsConn net.Conn
	tcpConn, currentEndpoint.TCPConnectMeasurement = TCPDo(ctx, endpoint)
	if tcpConn == nil {
		return currentEndpoint
	}
	defer tcpConn.Close() // suboptimal of course

	tlsConn, currentEndpoint.TLSHandshakeMeasurement = TLSDo(tcpConn, rt.Request.URL.Hostname())
	if tlsConn == nil {
		return currentEndpoint
	}
	defer tlsConn.Close() // suboptimal of course

	transport := nwebconnectivity.NewSingleTransport(tlsConn)
	currentEndpoint.HTTPRoundtripMeasurement = HTTPDo(rt.Request, transport)
	return currentEndpoint
}

func GenerateH3Endpoint(ctx context.Context, rt *RoundTrip, endpoint string) EndpointMeasurement {
	currentEndpoint := &H3EndpointMeasurement{
		Endpoint: endpoint,
	}
	var sess quic.EarlySession
	tlsConf := &tls.Config{
		ServerName: rt.Request.URL.Hostname(),
		NextProtos: []string{rt.proto},
	}
	sess, currentEndpoint.QUICHandshakeMeasurement = QUICDo(ctx, endpoint, tlsConf)
	if sess == nil {
		return currentEndpoint
	}
	transport := nwebconnectivity.NewSingleH3Transport(sess, tlsConf, &quic.Config{})
	currentEndpoint.HTTPRoundtripMeasurement = HTTPDo(rt.Request, transport)

	return currentEndpoint
}
