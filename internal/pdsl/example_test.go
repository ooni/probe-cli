package pdsl_test

import (
	"context"
	"crypto/tls"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/pdsl"
)

// This example shows how to create a typical measurement pipeline.
func Example() {
	// create a logger equivalent to miniooni's logger
	log.Log = &log.Logger{Level: log.InfoLevel, Handler: logx.NewHandlerWithDefaultSettings()}

	ctx := context.Background()

	// create a minimal runtime and make sure we eventually close all connections
	rt := pdsl.NewMinimalRuntime(log.Log)
	defer rt.Close()

	// create a suitable TLS configuration
	tlsConfig := &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
		ServerName: "www.example.com",
		RootCAs:    nil, // use netxlite default CA
	}

	// resolve IP addresses using two resolvers and deduplicate the results
	ipAddrs := pdsl.DNSLookupDeduplicate()(
		pdsl.Merge(
			pdsl.DNSLookupGetaddrinfo(ctx, rt)("www.example.com"),
			pdsl.DNSLookupUDP(ctx, rt, "8.8.8.8:53")("www.example.com"),
		),
	)

	// convert the IP addresses to endpoints
	endpoints := pdsl.MakeEndpointsForPort("443")(ipAddrs)

	// create TCP connections from endpoints using a goroutine pool
	conns := pdsl.Merge(pdsl.Fork(4, pdsl.TCPConnect(ctx, rt), endpoints)...)

	// create TLS connections also using a goroutine pool
	tlsConns := pdsl.Merge(pdsl.Fork(2, pdsl.TLSHandshake(ctx, rt, tlsConfig), conns)...)

	// make sure we run until completion
	_ = pdsl.Collect(tlsConns)
}
