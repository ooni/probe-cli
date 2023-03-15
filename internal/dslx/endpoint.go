package dslx

//
// Manipulate endpoints
//

import (
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
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

	// IDGenerator is MANDATORY the ID generator to use.
	IDGenerator *atomic.Int64

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// Network is the MANDATORY endpoint network.
	Network string

	// ZeroTime is the MANDATORY zero time of the measurement.
	ZeroTime time.Time
}

// EndpointOption is an option you can use to construct EndpointState.
type EndpointOption func(*Endpoint)

// EndpointOptionDomain allows to set the domain.
func EndpointOptionDomain(value string) EndpointOption {
	return func(es *Endpoint) {
		es.Domain = value
	}
}

// EndpointOptionIDGenerator allows to set the ID generator.
func EndpointOptionIDGenerator(value *atomic.Int64) EndpointOption {
	return func(es *Endpoint) {
		es.IDGenerator = value
	}
}

// EndpointOptionLogger allows to set the logger.
func EndpointOptionLogger(value model.Logger) EndpointOption {
	return func(es *Endpoint) {
		es.Logger = value
	}
}

// EndpointOptionZeroTime allows to set the zero time.
func EndpointOptionZeroTime(value time.Time) EndpointOption {
	return func(es *Endpoint) {
		es.ZeroTime = value
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
		Address:     string(address),
		Domain:      "",
		IDGenerator: &atomic.Int64{},
		Logger:      model.DiscardLogger,
		Network:     string(network),
		ZeroTime:    time.Now(),
	}
	for _, option := range options {
		option(epnt)
	}
	return epnt
}
