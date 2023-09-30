package dslnet

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/dslmodel"
	"github.com/ooni/probe-cli/v3/internal/logx"
)

// Getaddrinfo resolves domain names using getaddrinfo(3).
func Getaddrinfo(ctx context.Context, rt dslmodel.Runtime, query DNSQuery) ([]Endpoint, error) {
	// start the operation logger
	traceID := rt.NewTraceID()
	ol := logx.NewOperationLogger(rt.Logger(), "trace#%d: Getaddrinfo %s", traceID, query.Domain)

	// create trace for collecting OONI observations
	trace := rt.NewTrace(traceID, rt.ZeroTime(), query.Tags...)

	// perform the getaddrinfo(3) lookup
	resolver := trace.NewStdlibResolver(rt.Logger())
	addrs, err := resolver.LookupHost(ctx, query.Domain)

	// stop the operation logger
	ol.Stop(err)

	// handle error case
	if err != nil {
		return nil, err
	}

	// handle successful case
	outputs := []Endpoint{}
	for _, addr := range addrs {
		epnt := query.EndpointTemplate.Clone()
		epnt.Domain = query.Domain
		epnt.IPAddress = addr
		outputs = append(outputs, epnt)
	}
	return outputs, nil
}

// GetaddrinfoPipeline returns a [dslmodel.Pipeline] that calls [Getaddrinfo].
func GetaddrinfoPipeline() dslmodel.Pipeline[DNSQuery, Endpoint] {
	return dslmodel.GeneratorToPipeline(dslmodel.AsyncOperationToGenerator(
		dslmodel.FunctionWithSliceResultToAsyncOperation(Getaddrinfo),
	))
}
