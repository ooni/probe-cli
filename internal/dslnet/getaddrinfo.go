package dslnet

import (
	"context"
	"time"

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

	// enforce a timeout
	const timeout = 4 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// perform the getaddrinfo(3) lookup
	resolver := trace.NewStdlibResolver(rt.Logger())
	addrs, err := resolver.LookupHost(ctx, query.Domain)

	// stop the operation logger
	ol.Stop(err)

	// return to the caller
	return dnsUtilReturn(query, addrs, err)
}

// GetaddrinfoPipeline returns a [dslmodel.Pipeline] that calls [Getaddrinfo].
func GetaddrinfoPipeline() dslmodel.Pipeline[DNSQuery, Endpoint] {
	return dslmodel.GeneratorToPipeline(dslmodel.AsyncOperationToGenerator(
		dslmodel.FunctionWithSliceResultToAsyncOperation(Getaddrinfo),
	))
}
