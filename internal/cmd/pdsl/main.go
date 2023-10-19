package main

import (
	"context"
	"crypto/tls"
	"os"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/pdsl"
)

const udpResolverEndpoint = pdsl.Endpoint("8.8.8.8:53")

func dnsExperiment(ctx context.Context, rt pdsl.Runtime, domain pdsl.DomainName) []pdsl.Result[pdsl.IPAddr] {
	// resolve IP addresses with two parallel DNS resolvers
	return pdsl.Collect(pdsl.DNSLookupDedup()(pdsl.Merge(
		pdsl.DNSLookupGetaddrinfo(ctx, rt)(domain),
		pdsl.DNSLookupUDP(ctx, rt, udpResolverEndpoint)(domain),
	)))
}

func httpExperiment(ctx context.Context, rt pdsl.Runtime, domain pdsl.DomainName,
	ipAddrs ...pdsl.Result[pdsl.IPAddr]) <-chan pdsl.Result[pdsl.TCPConn] {
	// convert the IP addresses to endpoints
	endpoints := pdsl.MakeEndpointsForPort("80")(pdsl.Stream(ipAddrs...))

	// create TCP connections from endpoints using a goroutine pool
	return pdsl.Merge(pdsl.Fork(4, pdsl.TCPConnect(ctx, rt), endpoints)...)
}

func httpsExperiment(ctx context.Context, rt pdsl.Runtime, domain pdsl.DomainName,
	ipAddrs ...pdsl.Result[pdsl.IPAddr]) <-chan pdsl.Result[pdsl.TLSConn] {
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
	return pdsl.Merge(pdsl.Fork(2, pdsl.TLSHandshake(ctx, rt, tlsConfig), conns)...)
}

func mainExperiment(ctx context.Context, rt pdsl.Runtime, domain pdsl.DomainName) {
	// run the DNS experiment until completion
	ipAddrs := dnsExperiment(ctx, rt, domain)

	// TODO: invoke the TH

	// start the HTTPS experiment
	tlsConns := httpsExperiment(ctx, rt, domain, ipAddrs...)

	// start the HTTP experiment
	tcpConns := httpExperiment(ctx, rt, domain, ipAddrs...)

	// wait for both experiments to terminate
	_ = pdsl.Collect(
		pdsl.Discard[pdsl.TLSConn]()(tlsConns),
		pdsl.Discard[pdsl.TCPConn]()(tcpConns),
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
