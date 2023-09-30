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

	// TODO(bassosimone): we need to reduce duplicate addresses
	// TODO(bassosimone): we need to mix multiple resolvers results

	pipeline := dslmodel.JoinPipeline(
		dslnet.GetaddrinfoPipeline(),
		dslmodel.Parallel(4, dslnet.ConnectPipeline()),
	)

	input := dslnet.DNSQuery{
		Domain: "www.example.com",
		EndpointTemplate: dslnet.EndpointTemplate{
			Network: "tcp",
			Port:    "443",
			Tags:    []string{"endpoint"},
		},
		Tags: []string{"dns"},
	}

	outputs := pipeline.Run(ctx, rt, dslmodel.StreamResultValue(input))
	for output := range outputs {
		log.Infof("%+v", output)
	}
}
