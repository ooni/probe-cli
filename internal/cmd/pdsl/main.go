package main

import (
	"context"
	"crypto/tls"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/pdsl"
)

func main() {
	log.Log = &log.Logger{Level: log.InfoLevel, Handler: logx.NewHandlerWithDefaultSettings()}

	ctx := context.Background()
	rt := pdsl.NewMinimalRuntime(log.Log)

	domain := pdsl.DomainName("www.example.com")
	tlsConfig := &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
		ServerName: "www.example.com",
		RootCAs:    nil, // use netxlite default CA
	}

	// make sure we close all open connections
	defer rt.Close()

	// resolve IP addresses using two resolvers and deduplicate the results
	ipAddrs := pdsl.DNSLookupDeduplicate(pdsl.Merge(
		pdsl.DNSLookupGetaddrinfo(ctx, rt)(domain),
		pdsl.DNSLookupUDP(ctx, rt, pdsl.Endpoint("8.8.8.8:53"))(domain),
	))

	// convert the IP addresses to endpoints
	endpoints := pdsl.MakeEndpointsForPort("443")(ipAddrs)

	// create TCP connections from endpoints using a goroutine pool
	conns := pdsl.Merge(pdsl.Fork(4, pdsl.TCPConnect(ctx, rt), endpoints)...)

	// create TLS connections also using a goroutine pool
	tconns := pdsl.Merge(pdsl.Fork(2, pdsl.TLSHandshake(ctx, rt, tlsConfig), conns)...)

	// make sure we run until completion
	pdsl.Drain(tconns)
}
