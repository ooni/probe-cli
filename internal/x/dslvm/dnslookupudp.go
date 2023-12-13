package dslvm

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
)

// DNSLookupUDPStage is a [Stage] that resolves domain names using an UDP resolver.
type DNSLookupUDPStage struct {
	// Domain is the MANDATORY domain to resolve using this DNS resolver.
	Domain string

	// Output is the MANDATORY channel emitting IP addresses. We will close this
	// channel when we have finished streaming the resolved addresses.
	Output chan<- string

	// Resolver is the MANDATORY resolver endpoint (e.g., [::1]:53).
	Resolver string

	// Tags contains OPTIONAL tags for the DNS observations.
	Tags []string
}

var _ Stage = &DNSLookupUDPStage{}

// Run resolves a Domain using the given Do53 Endpoint and streams the
// results on Output, which is closed when we're done.
//
// This function honours the semaphore returned by the [Runtime] ActiveDNSLookups
// method and waits until it's given the permission to start a lookup.
func (sx *DNSLookupUDPStage) Run(ctx context.Context, rtx Runtime) {
	// wait for permission to lookup and signal when done
	rtx.ActiveDNSLookups().Wait()
	defer rtx.ActiveDNSLookups().Signal()

	// make sure we close output when done
	defer close(sx.Output)

	// create trace
	trace := rtx.NewTrace(rtx.IDGenerator().Add(1), rtx.ZeroTime(), sx.Tags...)

	// start operation logger
	ol := logx.NewOperationLogger(
		rtx.Logger(),
		"[#%d] DNSLookup[%s/udp] %s",
		trace.Index(),
		sx.Resolver,
		sx.Domain,
	)

	// setup
	const timeout = 4 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// create the resolver
	resolver := trace.NewParallelUDPResolver(
		rtx.Logger(),
		trace.NewDialerWithoutResolver(rtx.Logger()),
		sx.Resolver,
	)

	// lookup
	addrs, err := resolver.LookupHost(ctx, sx.Domain)

	// stop the operation logger
	ol.Stop(err)

	// save the observations
	rtx.SaveObservations(maybeTraceToObservations(trace)...)

	// handle error case
	if err != nil {
		return
	}

	// handle success
	for _, addr := range addrs {
		sx.Output <- addr
	}
}
