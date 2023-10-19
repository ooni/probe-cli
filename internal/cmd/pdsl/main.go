package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"os"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/pdsl"
)

const udpResolverEndpoint = pdsl.Endpoint("8.8.8.8:53")

func newRequest(scheme string, domain pdsl.DomainName, urlPath string) *http.Request {
	return &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: scheme,
			Host:   string(domain),
			Path:   urlPath,
		},
		Header: map[string][]string{},
		Host:   string(domain),
	}
}

func dnsExperiment(ctx context.Context, rt pdsl.Runtime, domain pdsl.DomainName) []pdsl.Result[pdsl.IPAddr] {
	// resolve IP addresses with two parallel DNS resolvers
	return pdsl.Collect(pdsl.DNSLookupDedup()(pdsl.Merge(
		pdsl.DNSLookupGetaddrinfo(ctx, rt)(domain),
		pdsl.DNSLookupUDP(ctx, rt, udpResolverEndpoint)(domain),
	)))
}

func httpExperiment(ctx context.Context, rt pdsl.Runtime, domain pdsl.DomainName,
	ipAddrs ...pdsl.Result[pdsl.IPAddr]) <-chan pdsl.Result[pdsl.HTTPResponse] {
	// convert the IP addresses to endpoints
	endpoints := pdsl.MakeEndpointsForPort("80")(pdsl.Stream(ipAddrs...))

	// create TCP connections from endpoints using a goroutine pool
	tcpConns := pdsl.Merge(pdsl.Fork(4, pdsl.TCPConnect(ctx, rt), endpoints)...)

	// create the request
	req := newRequest("http", domain, "/")

	// perform round trip with each connection
	return pdsl.HTTPRoundTripTCP(ctx, rt, req)(tcpConns)
}

func httpsExperiment(ctx context.Context, rt pdsl.Runtime, domain pdsl.DomainName,
	ipAddrs ...pdsl.Result[pdsl.IPAddr]) <-chan pdsl.Result[pdsl.HTTPResponse] {
	// create a suitable TLS configuration
	tlsConfig := &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
		ServerName: string(domain),
		RootCAs:    nil, // use netxlite default CA
	}

	// convert the IP addresses to endpoints
	endpoints := pdsl.MakeEndpointsForPort("443")(pdsl.Stream(ipAddrs...))

	// create TCP connections from endpoints using a goroutine pool
	conns := pdsl.Merge(pdsl.Fork(4, pdsl.TCPConnect(ctx, rt), endpoints)...)

	// create TLS connections also using a goroutine pool
	tlsConns := pdsl.Merge(pdsl.Fork(2, pdsl.TLSHandshake(ctx, rt, tlsConfig), conns)...)

	// create the request
	req := newRequest("https", domain, "/")

	// perform round trip with each connection
	return pdsl.HTTPRoundTripTLS(ctx, rt, req)(tlsConns)
}

func http3Experiment(ctx context.Context, rt pdsl.Runtime, domain pdsl.DomainName,
	ipAddrs ...pdsl.Result[pdsl.IPAddr]) <-chan pdsl.Result[pdsl.HTTPResponse] {
	// create a suitable TLS configuration
	tlsConfig := &tls.Config{
		NextProtos: []string{"h3"},
		ServerName: string(domain),
		RootCAs:    nil, // use netxlite default CA
	}

	// convert the IP addresses to endpoints
	endpoints := pdsl.MakeEndpointsForPort("443")(pdsl.Stream(ipAddrs...))

	// create QUIC connections also using a goroutine pool
	quicConns := pdsl.Merge(pdsl.Fork(2, pdsl.QUICHandshake(ctx, rt, tlsConfig), endpoints)...)

	// create the request
	req := newRequest("https", domain, "/")

	// perform round trip with each connection
	return pdsl.HTTPRoundTripQUIC(ctx, rt, req, tlsConfig)(quicConns)
}

func mainExperiment(ctx context.Context, rt pdsl.Runtime, domain pdsl.DomainName) {
	// run the DNS experiment until completion
	ipAddrs := dnsExperiment(ctx, rt, domain)

	// TODO: invoke the TH

	// wait for all the experiments to terminate
	_ = pdsl.Collect(
		httpsExperiment(ctx, rt, domain, ipAddrs...),
		httpExperiment(ctx, rt, domain, ipAddrs...),
		http3Experiment(ctx, rt, domain, ipAddrs...),
	)
}

func main() {
	log.Log = &log.Logger{Level: log.InfoLevel, Handler: logx.NewHandlerWithDefaultSettings()}

	ctx := context.Background()

	// create runtime and make sure we close open connections
	rt := pdsl.NewMinimalRuntime(log.Log)
	defer rt.Close()

	mainExperiment(ctx, rt, pdsl.DomainName(os.Args[1]))
}
