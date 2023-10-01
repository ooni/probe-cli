package main

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/dslminimalruntime"
	"github.com/ooni/probe-cli/v3/internal/dslmodel"
	"github.com/ooni/probe-cli/v3/internal/dslnet"
)

func main() {
	ctx := context.Background()

	rt := dslminimalruntime.New(log.Log)
	defer rt.Close()

	/*
		dnsQuery := dslnet.DNSQuery{
			Domain: "www.example.com",
			EndpointTemplate: dslnet.EndpointTemplate{
				Network: "tcp",
				Port:    "443",
				Tags:    []string{},
			},
			Tags: []string{},
		}

		dnsGetaddrinfo := dslnet.DNSLookupGetaddrinfo(dnsQuery)

		dnsUDP := dslnet.DNSLookupUDP("8.8.8.8:53", dnsQuery)

		dnsDedup := dslcore.Deduplicate(dnsGetaddrinfo, dnsUDP)

		tcpConn := dslmodel.FanOut(dslmodel.NumWorkers(4), dslnet.Connect(dnsDedup))

		tlsConn := dslmodel.FanOut(dslmodel.NumWorkers(4), dslnet.TLSHandshake(tcpConn))

		-----

		// resolve domain names using getaddrinfo
		dnsGetaddrinfo := dslnet.DNSLookupGetaddrinfo(ctx, rt, "www.example.com")

		// resolve domain names using DNS over UDP resolver
		dnsUDP := dnslnet.DNSLookupUDP("8.8.8.8:53")(ctx, rt, "www.example.com")

		// deduplicate IP addresses
		dnsUnique := dslcore.Dedup(dnsGetaddrinfo, dnsUDP)

		// convert unique IP addresses to endpoints
		epnts := dslnet.MakeEndpoints(dnsUnique, dslnet.EndpointPort("443"), dslnet.EndpointNetwork("tcp"))

		// establish TCP connections using parallel workers
		tcpConns := dslcore.FanOut(ctx, rt, dslmodel.NumWorkers(4), dslnet.Connector())

		// perform TLS handshakes using parallel workers
		tlsConns := dlscore.FanOut(ctx, rt, dslmodel.NumWorkers(4), dlsnet.TLSHandshaker(
			dslnet.TLSHandshakeALPN("h2", "http/1.1"),
			dslnet.TLSHandshakeSNI("www.example.com"),
		))

	*/

	pipeline := dslmodel.ComposePipelines3(
		dslmodel.Distribute(
			dslnet.GetaddrinfoPipeline(),
			dslnet.DNSLookupUDPPipeline("8.8.8.8:53"),
		),
		dslmodel.DedupPipeline[dslnet.Endpoint](),
		dslmodel.FanOut(
			dslmodel.NumWorkers(4),
			dslnet.ConnectPipeline(),
		),
	)

	input := dslnet.DNSQuery{
		Domain: "www.example.com",
		EndpointTemplate: dslnet.EndpointTemplate{
			Network: "tcp",
			Port:    "443",
			Tags:    []string{},
		},
		Tags: []string{},
	}

	outputs := pipeline.Run(ctx, rt, dslmodel.StreamResultValue(input))
	for output := range outputs {
		log.Infof("%+v", output)
	}
}
