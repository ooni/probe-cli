package dslx

//
// Manipulate endpoints
//

import (
	"context"
	"net"
	"strconv"
)

type (
	// EndpointNetwork is the network of the endpoint
	EndpointNetwork string

	// EndpointAddress is the endpoint address.
	EndpointAddress string
)

// Endpoint is a network endpoint along with configuration for measuring it. You
// should construct from an AddressSet or using NewEndpoint. Otherwise, make sure
// you initialize all the fields marked as MANDATORY.
type Endpoint struct {
	// Address is the MANDATORY endpoint address.
	Address string

	// Domain is the OPTIONAL domain used to resolve the endpoints' IP address.
	Domain string

	// Network is the MANDATORY endpoint network.
	Network string

	// Tags contains OPTIONAL tags for tagging observations.
	Tags []string
}

// EndpointOption is an option you can use to construct EndpointState.
type EndpointOption func(*Endpoint)

// EndpointOptionDomain allows to set the domain.
func EndpointOptionDomain(value string) EndpointOption {
	return func(es *Endpoint) {
		es.Domain = value
	}
}

// EndpointOptionTags allows to set tags to tag observations.
func EndpointOptionTags(value ...string) EndpointOption {
	return func(es *Endpoint) {
		es.Tags = append(es.Tags, value...)
	}
}

// NewEndpoint creates a new network endpoint (i.e., a three tuple composed
// of a network protocol, an IP address, and a port).
//
// Arguments:
//
// - network is either "tcp" or "udp";
//
// - address is the NewEndpoint address represented as an IP address followed by ":"
// followed by a port. IPv6 addresses must be quoted (e.g., "[::1]:80");
//
// - options contains additional options.
func NewEndpoint(
	network EndpointNetwork, address EndpointAddress, options ...EndpointOption) *Endpoint {
	epnt := &Endpoint{
		Address: string(address),
		Domain:  "",
		Network: string(network),
		Tags:    []string{},
	}
	for _, option := range options {
		option(epnt)
	}
	return epnt
}

// MakeEndpoint returns a [Func] that creates an [*Endpoint] given [*ResolvedAddress].
func MakeEndpoint(network EndpointNetwork, port EndpointPort, options ...EndpointOption) Func[*ResolvedAddress, *Endpoint] {
	return Operation[*ResolvedAddress, *Endpoint](func(ctx context.Context, addr *ResolvedAddress) (*Endpoint, error) {
		// create the destination endpoint address
		addrport := EndpointAddress(net.JoinHostPort(addr.Address, strconv.Itoa(int(port))))

		// make sure we include the proper domain name first but allow the caller
		// to potentially override the domain name using options
		allOptions := []EndpointOption{
			EndpointOptionDomain(addr.Domain),
		}
		allOptions = append(allOptions, options...)

		// build and return the endpoint
		endpoint := NewEndpoint(network, addrport, allOptions...)
		return endpoint, nil
	})
}

// MeasureResolvedAddresses returns a [Func] that measures the resolved addresses provided
// as the input argument using each of the provided functions.
func MeasureResolvedAddresses(fxs ...Func[*ResolvedAddress, Void]) Func[*ResolvedAddresses, Void] {
	return Operation[*ResolvedAddresses, Void](func(ctx context.Context, addrs *ResolvedAddresses) (Void, error) {
		// TODO(https://github.com/ooni/probe/issues/2619): we may want to configure this
		const parallelism = Parallelism(3)

		// run the matrix until the output is drained
		for range Matrix(ctx, parallelism, addrs.Flatten(), fxs) {
			// nothing
		}

		return Void{}, nil
	})
}
