package dslx

//
// Manipulate endpoints
//

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
