package main

import (
	"context"
	"crypto/tls"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/pdsl"
)

func measureWithTLS(ctx context.Context, rt pdsl.Runtime,
	domain pdsl.DomainName, udpResolverEndpoint pdsl.Endpoint) <-chan pdsl.Result[pdsl.Void] {
	// create a suitable TLS configuration
	tlsConfig := &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
		ServerName: string(domain),
		RootCAs:    nil, // use netxlite default CA
	}

	// resolve IP addresses using two resolvers and deduplicate the results
	ipAddrs := pdsl.Collect(pdsl.DNSLookupDedup()(pdsl.Merge(
		pdsl.DNSLookupGetaddrinfo(ctx, rt)(domain),
		pdsl.DNSLookupUDP(ctx, rt, udpResolverEndpoint)(domain),
	)))

	// convert the IP addresses to endpoints
	endpoints := pdsl.MakeEndpointsForPort("443")(pdsl.Stream(ipAddrs...))

	// create TCP connections from endpoints using a goroutine pool
	conns := pdsl.Merge(pdsl.Fork(4, pdsl.TCPConnect(ctx, rt), endpoints)...)

	// create TLS connections also using a goroutine pool
	tlsConns := pdsl.Merge(pdsl.Fork(2, pdsl.TLSHandshake(ctx, rt, tlsConfig), conns)...)

	// return a list of void
	return pdsl.Discard[pdsl.TLSConn]()(tlsConns)
}

func main() {
	log.Log = &log.Logger{Level: log.InfoLevel, Handler: logx.NewHandlerWithDefaultSettings()}

	ctx := context.Background()

	// create runtime and make sure we close open connections
	rt := pdsl.NewMinimalRuntime(log.Log)
	defer rt.Close()

	_ = pdsl.Collect(
		measureWithTLS(ctx, rt, "www.example.com", "8.8.8.8:53"),
		measureWithTLS(ctx, rt, "www.youtube.com", "8.8.8.8:53"),
	)
}
