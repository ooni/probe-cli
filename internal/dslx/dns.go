package dslx

//
// DNS measurements
//

import (
	"context"
	"errors"
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
}

// Flatten transforms a [ResolvedAddresses] into a slice of zero or more [ResolvedAddress].
func (ra *ResolvedAddresses) Flatten() (out []*ResolvedAddress) {
	for _, ipAddr := range ra.Addresses {
		out = append(out, &ResolvedAddress{
			Address: ipAddr,
			Domain:  ra.Domain,
		})
	}
	return
}

// ResolvedAddress is a single address resolved using a DNS lookup function.
type ResolvedAddress struct {
	// Address is the address that was resolved.
	Address string

	// Domain is the domain from which we resolved the address.
	Domain string
}

// DNSLookupGetaddrinfo returns a function that resolves a domain name to
// IP addresses using libc's getaddrinfo function.
func DNSLookupGetaddrinfo(rt Runtime) Func[*DomainToResolve, *ResolvedAddresses] {
	return Operation[*DomainToResolve, *ResolvedAddresses](func(ctx context.Context, input *DomainToResolve) (*ResolvedAddresses, error) {
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

		// save the observations
		rt.SaveObservations(maybeTraceToObservations(trace)...)

		// handle error case
		if err != nil {
			ol.Stop(err)
			return nil, err
		}

		// handle success
		ol.Stop(addrs)
		state := &ResolvedAddresses{
			Addresses: addrs,
			Domain:    input.Domain,
		}
		return state, nil
	})
}

// DNSLookupUDP returns a function that resolves a domain name to
// IP addresses using the given DNS-over-UDP resolver.
func DNSLookupUDP(rt Runtime, endpoint string) Func[*DomainToResolve, *ResolvedAddresses] {
	return Operation[*DomainToResolve, *ResolvedAddresses](func(ctx context.Context, input *DomainToResolve) (*ResolvedAddresses, error) {
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

		// save the observations
		rt.SaveObservations(maybeTraceToObservations(trace)...)

		// handle error case
		if err != nil {
			ol.Stop(err)
			return nil, err
		}

		// handle success
		ol.Stop(addrs)
		state := &ResolvedAddresses{
			Addresses: addrs,
			Domain:    input.Domain,
		}
		return state, nil
	})
}

// ErrDNSLookupParallel indicates that DNSLookupParallel failed.
var ErrDNSLookupParallel = errors.New("dslx: DNSLookupParallel failed")

// DNSLookupParallel runs DNS lookups in parallel. On success, this function returns
// a unique list of IP addresses aggregated from all resolvers. On failure, this function
// returns [ErrDNSLookupParallel]. You can always obtain the individual errors by
// processing observations or by creating a per-DNS-resolver pipeline.
func DNSLookupParallel(fxs ...Func[*DomainToResolve, *ResolvedAddresses]) Func[*DomainToResolve, *ResolvedAddresses] {
	return Operation[*DomainToResolve, *ResolvedAddresses](func(ctx context.Context, domain *DomainToResolve) (*ResolvedAddresses, error) {
		// TODO(bassosimone): we may want to configure this
		const parallelism = Parallelism(3)

		// run all the DNS resolvers in parallel
		results := Parallel(ctx, parallelism, domain, fxs...)

		// reduce addresses
		addressSet := NewAddressSet()
		for _, result := range results {
			if err := result.Error; err != nil {
				continue
			}
			addressSet.Add(result.State.Addresses...)
		}
		uniq := addressSet.Uniq()

		// handle the case where all the DNS resolvers failed
		if len(uniq) < 1 {
			return nil, ErrDNSLookupParallel
		}

		// handle success
		state := &ResolvedAddresses{
			Addresses: uniq,
			Domain:    domain.Domain,
		}
		return state, nil
	})
}
