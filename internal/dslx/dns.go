package dslx

//
// DNS measurements
//

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// DomainName is a domain name to resolve.
type DomainName string

// DNSLookupOption is an option you can pass to NewDomainToResolve.
type DNSLookupOption func(*DomainToResolve)

// DNSLookupOptionIDGenerator configures a specific ID generator.
// See DomainToResolve docs for more information.
func DNSLookupOptionIDGenerator(value *atomic.Int64) DNSLookupOption {
	return func(dis *DomainToResolve) {
		dis.IDGenerator = value
	}
}

// DNSLookupOptionLogger configures a specific logger.
// See DomainToResolve docs for more information.
func DNSLookupOptionLogger(value model.Logger) DNSLookupOption {
	return func(dis *DomainToResolve) {
		dis.Logger = value
	}
}

// DNSLookupOptionZeroTime configures the measurement's zero time.
// See DomainToResolve docs for more information.
func DNSLookupOptionZeroTime(value time.Time) DNSLookupOption {
	return func(dis *DomainToResolve) {
		dis.ZeroTime = value
	}
}

// NewDomainToResolve creates input for performing DNS lookups. The only mandatory
// argument is the domain name to resolve. You can also supply optional
// values by passing options to this function.
func NewDomainToResolve(domain DomainName, options ...DNSLookupOption) *DomainToResolve {
	state := &DomainToResolve{
		Domain:      string(domain),
		IDGenerator: &atomic.Int64{},
		Logger:      model.DiscardLogger,
		ZeroTime:    time.Now(),
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

	// IDGenerator is the MANDATORY ID generator. We will use this field
	// to assign unique IDs to distinct sub-measurements. The default
	// construction implemented by NewDomainToResolve creates a new generator
	// that starts counting from zero, leading to the first trace having
	// one as its index.
	IDGenerator *atomic.Int64

	// Logger is the MANDATORY logger to use. The default construction
	// implemented by NewDomainToResolve uses model.DiscardLogger.
	Logger model.Logger

	// ZeroTime is the MANDATORY zero time of the measurement. We will
	// use this field as the zero value to compute relative elapsed times
	// when generating measurements. The default construction by
	// NewDomainToResolve initializes this field with the current time.
	ZeroTime time.Time
}

// ResolvedAddresses is the contains the results of DNS lookups. To initialize
// this struct manually, follow specific instructions for each field.
type ResolvedAddresses struct {
	// Addresses contains the nonempty resolved addresses.
	Addresses []string

	// Domain is the domain we resolved. We inherit this field
	// from the value inside the DomainToResolve.
	Domain string

	// IDGenerator is the ID generator. We inherit this field
	// from the value inside the DomainToResolve.
	IDGenerator *atomic.Int64

	// Logger is the logger to use. We inherit this field
	// from the value inside the DomainToResolve.
	Logger model.Logger

	// Trace is the trace we're currently using. This struct is
	// created by the various Apply functions using values inside
	// the DomainToResolve to initialize the Trace.
	Trace *measurexlite.Trace

	// ZeroTime is the zero time of the measurement. We inherit this field
	// from the value inside the DomainToResolve.
	ZeroTime time.Time
}

// DNSLookupGetaddrinfo returns a function that resolves a domain name to
// IP addresses using libc's getaddrinfo function.
func DNSLookupGetaddrinfo() Func[*DomainToResolve, *Maybe[*ResolvedAddresses]] {
	return &dnsLookupGetaddrinfoFunc{}
}

// dnsLookupGetaddrinfoFunc is the function returned by DNSLookupGetaddrinfo.
type dnsLookupGetaddrinfoFunc struct {
	resolver *mocks.Resolver // for testing
}

// Apply implements Func.
func (f *dnsLookupGetaddrinfoFunc) Apply(
	ctx context.Context, input *DomainToResolve) *Maybe[*ResolvedAddresses] {

	// create trace
	trace := measurexlite.NewTrace(input.IDGenerator.Add(1), input.ZeroTime)

	// start the operation logger
	ol := measurexlite.NewOperationLogger(
		input.Logger,
		"[#%d] DNSLookup[getaddrinfo] %s",
		trace.Index,
		input.Domain,
	)

	// setup
	const timeout = 4 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var resolver model.Resolver = f.resolver
	if resolver == nil {
		resolver = trace.NewStdlibResolver(input.Logger)
	}

	// lookup
	addrs, err := resolver.LookupHost(ctx, input.Domain)

	// stop the operation logger
	ol.Stop(err)

	state := &ResolvedAddresses{
		Addresses:   addrs, // maybe empty
		Domain:      input.Domain,
		IDGenerator: input.IDGenerator,
		Logger:      input.Logger,
		Trace:       trace,
		ZeroTime:    input.ZeroTime,
	}

	return &Maybe[*ResolvedAddresses]{
		Error:        err,
		Observations: maybeTraceToObservations(trace),
		Skipped:      false,
		State:        state,
	}
}

// DNSLookupUDP returns a function that resolves a domain name to
// IP addresses using the given DNS-over-UDP resolver.
func DNSLookupUDP(resolver string) Func[*DomainToResolve, *Maybe[*ResolvedAddresses]] {
	return &dnsLookupUDPFunc{
		Resolver: resolver,
	}
}

// dnsLookupUDPFunc is the function returned by DNSLookupUDP.
type dnsLookupUDPFunc struct {
	// Resolver is the MANDATORY resolver to use.
	Resolver string
}

// Apply implements Func.
func (f *dnsLookupUDPFunc) Apply(
	ctx context.Context, input *DomainToResolve) *Maybe[*ResolvedAddresses] {

	// create trace
	trace := measurexlite.NewTrace(input.IDGenerator.Add(1), input.ZeroTime)

	// start the operation logger
	ol := measurexlite.NewOperationLogger(
		input.Logger,
		"[#%d] DNSLookup[%s/udp] %s",
		trace.Index,
		f.Resolver,
		input.Domain,
	)

	// setup
	const timeout = 4 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resolver := trace.NewParallelUDPResolver(
		input.Logger,
		netxlite.NewDialerWithoutResolver(input.Logger),
		f.Resolver,
	)

	// lookup
	addrs, err := resolver.LookupHost(ctx, input.Domain)

	// stop the operation logger
	ol.Stop(err)

	state := &ResolvedAddresses{
		Addresses:   addrs, // maybe empty
		Domain:      input.Domain,
		IDGenerator: input.IDGenerator,
		Logger:      input.Logger,
		Trace:       trace,
		ZeroTime:    input.ZeroTime,
	}

	return &Maybe[*ResolvedAddresses]{
		Error:        err,
		Observations: maybeTraceToObservations(trace),
		Skipped:      false,
		State:        state,
	}
}
