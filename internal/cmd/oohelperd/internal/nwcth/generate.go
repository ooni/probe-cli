package nwcth

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/nwebconnectivity"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Generate is the third step of the algorithm. Given the
// observed round trips, we generate measurement targets and
// execute those measurements so the probe has a benchmark.

// Generator is the interface responsible for running Generate.
type Generator interface {
	Generate(ctx context.Context, rts []*RoundTrip) ([]*URLMeasurement, error)
}

// defaultGenerator is the default Generator.
type defaultGenerator struct {
	resolver netxlite.Resolver
}

// Generate takes in input a list of round trips and outputs
// a list of connectivity measurements for each of them.
func (g *defaultGenerator) Generate(ctx context.Context, rts []*RoundTrip) ([]*URLMeasurement, error) {
	var out []*URLMeasurement
	for _, rt := range rts {
		addrs, err := DNSDo(ctx, rt.Request.URL.Hostname(), g.resolver)
		currentURL := &URLMeasurement{
			DNS: &DNSMeasurement{
				Domain:  rt.Request.URL.Hostname(),
				Addrs:   addrs,
				Failure: newfailure(err),
			},
			RoundTrip: rt,
			URL:       rt.Request.URL.String(),
		}
		out = append(out, currentURL)
		if err != nil {
			return out, err
		}
		for _, addr := range addrs {
			var port string
			explicitPort := rt.Request.URL.Port()
			scheme := rt.Request.URL.Scheme
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
			endpoint := net.JoinHostPort(addr, port)

			var currentEndpoint EndpointMeasurement
			_, h3 := supportedQUICVersions[rt.proto]
			switch {
			case h3:
				currentEndpoint = g.GenerateH3Endpoint(ctx, rt, endpoint)
			case rt.proto == "http":
				currentEndpoint = g.GenerateHTTPEndpoint(ctx, rt, endpoint)
			case rt.proto == "https":
				currentEndpoint = g.GenerateHTTPSEndpoint(ctx, rt, endpoint)
			default:
				// TODO(kelmenhorst): do we have to register this error somewhere in the result struct?
				continue
			}
			currentURL.Endpoints = append(currentURL.Endpoints, currentEndpoint)
		}
	}
	return out, nil
}

func (g *defaultGenerator) GenerateHTTPEndpoint(ctx context.Context, rt *RoundTrip, endpoint string) EndpointMeasurement {
	currentEndpoint := &HTTPEndpointMeasurement{
		Endpoint: endpoint,
	}
	tcpConn, err := TCPDo(ctx, endpoint, newDialerResolver(g.resolver))
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

func (g *defaultGenerator) GenerateHTTPSEndpoint(ctx context.Context, rt *RoundTrip, endpoint string) EndpointMeasurement {
	currentEndpoint := &HTTPSEndpointMeasurement{
		Endpoint: endpoint,
	}
	var tcpConn, tlsConn net.Conn
	tcpConn, err := TCPDo(ctx, endpoint, newDialerResolver(g.resolver))
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

func (g *defaultGenerator) GenerateH3Endpoint(ctx context.Context, rt *RoundTrip, endpoint string) EndpointMeasurement {
	currentEndpoint := &H3EndpointMeasurement{
		Endpoint: endpoint,
	}
	tlsConf := &tls.Config{
		ServerName: rt.Request.URL.Hostname(),
		NextProtos: []string{rt.proto},
	}
	sess, err := QUICDo(ctx, endpoint, tlsConf, newQUICDialerResolver(g.resolver))
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
