package dslnet

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/dslmodel"
	"github.com/ooni/probe-cli/v3/internal/logx"
)

// DNSLookupUDP returns a func that resolves domain names using the given DNS-over-UDP endpoint.
func DNSLookupUDP(endpoint string) dslmodel.FunctionWithSliceResult[DNSQuery, Endpoint] {
	return func(ctx context.Context, rt dslmodel.Runtime, query DNSQuery) ([]Endpoint, error) {
		// start the operation logger
		traceID := rt.NewTraceID()
		ol := logx.NewOperationLogger(rt.Logger(), "trace#%d: DNSLookupUDP[%s] %s", traceID, endpoint, query.Domain)

		// create trace for collecting OONI observations
		trace := rt.NewTrace(traceID, rt.ZeroTime(), query.Tags...)

		// enforce a timeout
		const timeout = 4 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// perform a DNS-over-UDP lookup
		resolver := trace.NewParallelUDPResolver(rt.Logger(),
			trace.NewDialerWithoutResolver(rt.Logger()), endpoint)
		addrs, err := resolver.LookupHost(ctx, query.Domain)

		// stop the operation logger
		ol.Stop(err)

		// return to the caller
		return dnsUtilReturn(query, addrs, err)
	}
}

// DNSLookupUDPPipeline returns a [dslmodel.Pipeline] that calls [DNSLookupUDP].
func DNSLookupUDPPipeline(endpoint string) dslmodel.Pipeline[DNSQuery, Endpoint] {
	return dslmodel.GeneratorToPipeline(dslmodel.AsyncOperationToGenerator(
		dslmodel.FunctionWithSliceResultToAsyncOperation(DNSLookupUDP(endpoint)),
	))
}
