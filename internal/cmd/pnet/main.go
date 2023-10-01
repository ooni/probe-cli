package main

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/pnet"
)

func main() {
	ctx := context.Background()

	// create the pipeline
	pipeline := pnet.Compose3(
		pnet.Generic(pnet.Getaddrinfo()),
		pnet.Generic(pnet.Parallel(16, pnet.Connect())),
		pnet.Generic(pnet.Close[pnet.NetConn]()),
	)

	// create the input
	input := pnet.DNSQuery{
		Domain:          "www.youtube.com",
		EndpointNetwork: "tcp",
		EndpointPort:    "443",
		Logger:          log.Log,
	}

	// collect the results
	outputs := pnet.Run(ctx, pipeline, any(input))

	// print the results
	for _, entry := range outputs {
		log.Infof("output: %+v", entry)
	}
}
