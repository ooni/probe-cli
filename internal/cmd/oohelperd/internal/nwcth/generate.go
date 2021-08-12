package nwcth

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/nwebconnectivity"
)

// Generate is the third step of the algorithm. Given the
// observed round trips, we generate measurement targets and
// execute those measurements so the probe has a benchmark.

// Generate takes in input a list of round trips and outputs
// a list of connectivity measurements for each of them.
func Generate(ctx context.Context, rts []*RoundTrip) ([]*URLMeasurement, error) {
	var out []*URLMeasurement
	for _, rt := range rts {
		addrs, err := DNSDo(ctx, rt.Request.URL.Hostname(), newResolver())
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
	tcpConn, err := TCPDo(ctx, endpoint, newDialer())
	currentEndpoint.TCPConnectMeasurement = &TCPConnectMeasurement{
		Failure: newfailure(err),
	}
	if err != nil {
		return currentEndpoint
	}
	defer tcpConn.Close() // suboptimal of course

	transport := nwebconnectivity.NewSingleTransport(tcpConn)
	currentEndpoint.HTTPRoundtripMeasurement = HTTPDo(rt.Request, transport)
	return currentEndpoint
}

func GenerateHTTPSEndpoint(ctx context.Context, rt *RoundTrip, endpoint string) EndpointMeasurement {
	currentEndpoint := &HTTPSEndpointMeasurement{
		Endpoint: endpoint,
	}
	var tcpConn, tlsConn net.Conn
	tcpConn, err := TCPDo(ctx, endpoint, newDialer())
	currentEndpoint.TCPConnectMeasurement = &TCPConnectMeasurement{
		Failure: newfailure(err),
	}
	if err != nil {
		return currentEndpoint
	}
	defer tcpConn.Close() // suboptimal of course

	tlsConn, err = TLSDo(tcpConn, rt.Request.URL.Hostname())
	currentEndpoint.TLSHandshakeMeasurement = &TLSHandshakeMeasurement{
		Failure: newfailure(err),
	}
	if err != nil {
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
	tlsConf := &tls.Config{
		ServerName: rt.Request.URL.Hostname(),
		NextProtos: []string{rt.proto},
	}
	sess, err := QUICDo(ctx, endpoint, tlsConf, newQUICDialer())
	currentEndpoint.QUICHandshakeMeasurement = &TLSHandshakeMeasurement{
		Failure: newfailure(err),
	}
	if err != nil {
		return currentEndpoint
	}
	transport := nwebconnectivity.NewSingleH3Transport(sess, tlsConf, &quic.Config{})
	currentEndpoint.HTTPRoundtripMeasurement = HTTPDo(rt.Request, transport)

	return currentEndpoint
}
