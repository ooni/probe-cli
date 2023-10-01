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
		ServerName: string(domain),
		RootCAs:    nil, // use netxlite default CA
	}

	ipAddrsGetaddrinfo := pdsl.DNSLookupGetaddrinfo(ctx, rt)(domain)
	ipAddrs8888 := pdsl.DNSLookupUDP(ctx, rt, pdsl.Endpoint("8.8.8.8:53"))(domain)
	ipAddrsUnique := pdsl.DNSLookupDeduplicate(pdsl.Merge(ipAddrsGetaddrinfo, ipAddrs8888))
	endpoints := pdsl.MakeEndpointsForPort("443")(ipAddrsUnique)
	conns := pdsl.Merge(pdsl.Fork(4, pdsl.TCPConnect(ctx, rt), endpoints)...)
	tconns := pdsl.Merge(pdsl.Fork(2, pdsl.TLSHandshake(ctx, rt, tlsConfig), conns)...)

	for result := range tconns {
		log.Infof("%+v", result)
	}
}
