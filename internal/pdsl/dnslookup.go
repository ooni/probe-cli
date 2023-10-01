package pdsl

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
)

// DNSLookupGetaddrinfo creates a [Generator] that resolves domain names using getaddrinfo(3).
func DNSLookupGetaddrinfo(ctx context.Context, rt Runtime, tags ...string) Generator[DomainName, Result[IPAddr]] {
	return func(domain DomainName) <-chan Result[IPAddr] {
		// create the outputs channel
		outputs := make(chan Result[IPAddr])

		go func() {
			// make sure we close channel when done
			defer close(outputs)

			// start the operation logger
			traceID := rt.NewTraceID()
			ol := logx.NewOperationLogger(rt.Logger(), "[#%d] Getaddrinfo %s", traceID, domain)

			// create trace for collecting OONI observations
			trace := rt.NewTrace(traceID, rt.ZeroTime(), tags...)

			// enforce a timeout
			const timeout = 4 * time.Second
			ctx, cancel := context.WithTimeout(ctx, timeout)

			// perform the getaddrinfo(3) lookup
			resolver := trace.NewStdlibResolver(rt.Logger())
			addrs, err := resolver.LookupHost(ctx, string(domain))

			// cancel the context
			cancel()

			// stop the operation logger
			ol.Stop(err)

			// handle failure
			if err != nil {
				outputs <- NewResultError[IPAddr](err)
				return
			}

			// handle success
			for _, addr := range addrs {
				outputs <- NewResultValue(IPAddr(addr))
			}
		}()

		return outputs
	}
}

// DNSLookupUDP returns a [Generator] that resolves domain names to IP addresses using DNS-over-UDP.
func DNSLookupUDP(ctx context.Context, rt Runtime, endpoint Endpoint, tags ...string) Generator[DomainName, Result[IPAddr]] {
	return func(domain DomainName) <-chan Result[IPAddr] {
		// create the outputs channel
		outputs := make(chan Result[IPAddr])

		go func() {
			// make sure we close channel when done
			defer close(outputs)

			// start the operation logger
			traceID := rt.NewTraceID()
			ol := logx.NewOperationLogger(rt.Logger(), "[#%d] DNSLookupUDP[%s] %s", traceID, endpoint, domain)

			// create trace for collecting OONI observations
			trace := rt.NewTrace(traceID, rt.ZeroTime(), tags...)

			// enforce a timeout
			const timeout = 4 * time.Second
			ctx, cancel := context.WithTimeout(ctx, timeout)

			// perform a DNS-over-UDP lookup
			resolver := trace.NewParallelUDPResolver(rt.Logger(),
				trace.NewDialerWithoutResolver(rt.Logger()), string(endpoint))
			addrs, err := resolver.LookupHost(ctx, string(domain))

			// cancel the context
			cancel()

			// stop the operation logger
			ol.Stop(err)

			// handle failure
			if err != nil {
				outputs <- NewResultError[IPAddr](err)
				return
			}

			// handle success
			for _, addr := range addrs {
				outputs <- NewResultValue(IPAddr(addr))
			}
		}()

		return outputs
	}
}

// DNSLookupDeduplicate is a [Filter] that deduplicates the resolved IP addresses.
func DNSLookupDeduplicate(inputs <-chan Result[IPAddr]) <-chan Result[IPAddr] {
	// create channel for producing results
	outputs := make(chan Result[IPAddr])

	go func() {
		// make sure we close the channel when done
		defer close(outputs)

		// deduplicate addresses read over inputs
		already := make(map[IPAddr]bool)
		for input := range inputs {
			if err := input.Err; err != nil {
				outputs <- input
				continue
			}
			already[input.Value] = true
			outputs <- input
		}
	}()

	return outputs
}
