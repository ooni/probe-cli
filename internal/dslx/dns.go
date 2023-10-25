package dslx

//
// DNS measurements
//

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
)

// DomainName is a domain name to resolve.
type DomainName string

// DNSLookupOption is an option you can pass to NewDomainToResolve.
type DNSLookupOption func(*DomainToResolve)

// DNSLookupOptionTags allows to set tags to tag observations.
func DNSLookupOptionTags(value ...string) DNSLookupOption {
	return func(dis *DomainToResolve) {
		dis.Tags = append(dis.Tags, value...)
	}
}

// NewDomainToResolve creates input for performing DNS lookups. The only mandatory
// argument is the domain name to resolve. You can also supply optional
// values by passing options to this function.
func NewDomainToResolve(domain DomainName, options ...DNSLookupOption) *DomainToResolve {
	state := &DomainToResolve{
		Domain: string(domain),
		Tags:   []string{},
	}
	for _, option := range options {
		option(state)
	}
	return state
}

// DomainToResolve is the input for DNS lookup functions.
//
// You should construct this type using the NewDomainToResolve constructor
// as well as DNSLookupOption options to fill optional values. If you
// want to construct this type manually, please make sure you initialize
// all the variables marked as MANDATORY.
type DomainToResolve struct {
	// Domain is the MANDATORY domain name to lookup.
	Domain string

	// Tags contains OPTIONAL tags to tag observations.
	Tags []string
}

// ResolvedAddresses contains the results of DNS lookups. To initialize
// this struct manually, follow specific instructions for each field.
type ResolvedAddresses struct {
	// Addresses contains the nonempty resolved addresses.
	Addresses []string

	// Domain is the domain we resolved. We inherit this field
	// from the value inside the DomainToResolve.
	Domain string

	// Trace is the trace we're currently using. This struct is
	// created by the various Apply functions using values inside
	// the DomainToResolve to initialize the Trace.
	Trace Trace
}

// DNSLookupGetaddrinfo returns a function that resolves a domain name to
// IP addresses using libc's getaddrinfo function.
func DNSLookupGetaddrinfo(rt Runtime) Func[*DomainToResolve, *ResolvedAddresses] {
	return Operation[*DomainToResolve, *ResolvedAddresses](func(ctx context.Context, input *DomainToResolve) *Maybe[*ResolvedAddresses] {
		// create trace
		trace := rt.NewTrace(rt.IDGenerator().Add(1), rt.ZeroTime(), input.Tags...)

		// start the operation logger
		ol := logx.NewOperationLogger(
			rt.Logger(),
			"[#%d] DNSLookup[getaddrinfo] %s",
			trace.Index(),
			input.Domain,
		)

		// setup
		const timeout = 4 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// create the resolver
		resolver := trace.NewStdlibResolver(rt.Logger())

		// lookup
		addrs, err := resolver.LookupHost(ctx, input.Domain)

		// stop the operation logger
		ol.Stop(err)

		state := &ResolvedAddresses{
			Addresses: addrs, // maybe empty
			Domain:    input.Domain,
			Trace:     trace,
		}

		return &Maybe[*ResolvedAddresses]{
			Error:        err,
			Observations: maybeTraceToObservations(trace),
			State:        state,
		}
	})
}

// DNSLookupUDP returns a function that resolves a domain name to
// IP addresses using the given DNS-over-UDP resolver.
func DNSLookupUDP(rt Runtime, endpoint string) Func[*DomainToResolve, *ResolvedAddresses] {
	return Operation[*DomainToResolve, *ResolvedAddresses](func(ctx context.Context, input *DomainToResolve) *Maybe[*ResolvedAddresses] {
		// create trace
		trace := rt.NewTrace(rt.IDGenerator().Add(1), rt.ZeroTime(), input.Tags...)

		// start the operation logger
		ol := logx.NewOperationLogger(
			rt.Logger(),
			"[#%d] DNSLookup[%s/udp] %s",
			trace.Index(),
			endpoint,
			input.Domain,
		)

		// setup
		const timeout = 4 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// create the resolver
		resolver := trace.NewParallelUDPResolver(
			rt.Logger(),
			trace.NewDialerWithoutResolver(rt.Logger()),
			endpoint,
		)

		// lookup
		addrs, err := resolver.LookupHost(ctx, input.Domain)

		// stop the operation logger
		ol.Stop(err)

		state := &ResolvedAddresses{
			Addresses: addrs, // maybe empty
			Domain:    input.Domain,
			Trace:     trace,
		}

		return &Maybe[*ResolvedAddresses]{
			Error:        err,
			Observations: maybeTraceToObservations(trace),
			State:        state,
		}
	})
}
