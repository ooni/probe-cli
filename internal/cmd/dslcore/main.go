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
