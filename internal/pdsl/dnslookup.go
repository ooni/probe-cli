package pdsl

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
)

// DNSLookupGetaddrinfo creates a [Generator] that resolves domain names using getaddrinfo(3).
func DNSLookupGetaddrinfo(ctx context.Context, rt Runtime, tags ...string) Generator[DomainName, IPAddr] {
	return startGeneratorService(func(domainName DomainName) ([]IPAddr, error) {
		// start the operation logger
		traceID := rt.NewTraceID()
		ol := logx.NewOperationLogger(rt.Logger(), "[#%d] Getaddrinfo %s", traceID, domainName)

		// create trace for collecting OONI observations
		trace := rt.NewTrace(traceID, rt.ZeroTime(), tags...)

		// enforce a timeout
		const timeout = 4 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// perform the getaddrinfo(3) lookup
		resolver := trace.NewStdlibResolver(rt.Logger())
		addrs, err := resolver.LookupHost(ctx, string(domainName))

		// stop the operation logger
		ol.Stop(err)

		// handle failure
		if err != nil {
			return nil, err
		}

		// handle success
		return dnsNewIPAddrList(addrs), nil
	})
}

// DNSLookupUDP returns a [Generator] that resolves domain names to IP addresses using DNS-over-UDP.
func DNSLookupUDP(ctx context.Context, rt Runtime, endpoint Endpoint, tags ...string) Generator[DomainName, IPAddr] {
	return startGeneratorService(func(domainName DomainName) ([]IPAddr, error) {
		// start the operation logger
		traceID := rt.NewTraceID()
		ol := logx.NewOperationLogger(rt.Logger(), "[#%d] DNSLookupUDP[%s] %s", traceID, endpoint, domainName)

		// create trace for collecting OONI observations
		trace := rt.NewTrace(traceID, rt.ZeroTime(), tags...)

		// enforce a timeout
		const timeout = 4 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// perform a DNS-over-UDP lookup
		resolver := trace.NewParallelUDPResolver(rt.Logger(),
			trace.NewDialerWithoutResolver(rt.Logger()), string(endpoint))
		addrs, err := resolver.LookupHost(ctx, string(domainName))

		// stop the operation logger
		ol.Stop(err)

		// handle failure
		if err != nil {
			return nil, err
		}

		// handle success
		return dnsNewIPAddrList(addrs), nil
	})
}

func dnsNewIPAddrList(inputs []string) (outputs []IPAddr) {
	for _, input := range inputs {
		outputs = append(outputs, IPAddr(input))
	}
	return
}

// DNSLookupDedup returns a [Filter] that deduplicates the resolved IP addresses.
func DNSLookupDedup() Filter[IPAddr, IPAddr] {
	return func(inputs <-chan Result[IPAddr]) <-chan Result[IPAddr] {
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
				if already[input.Value] {
					continue
				}
				already[input.Value] = true
				outputs <- input
			}
		}()

		return outputs
	}
}
