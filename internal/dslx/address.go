package dslx

//
// Manipulate sets of IP addresses
//

import (
	"net"
	"strconv"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewAddressSet creates a new address set from optional addresses resolved by DNS.
func NewAddressSet(dns ...*Maybe[*ResolvedAddresses]) *AddressSet {
	uniq := make(map[string]bool)
	for _, e := range dns {
		if e.Skipped || e.Error != nil {
			continue
		}
		v := e.State
		for _, a := range v.Addresses {
			uniq[a] = true
		}
	}
	return &AddressSet{uniq}
}

// AddressSet is a set of IP addresses. The zero value struct
// is invalid, please initialize M or use NewAddressSet.
type AddressSet struct {
	// M is the map we use to represent the set.
	M map[string]bool
}

// Add MUTATES the set to add a (possibly-new) address to the set.
func (as *AddressSet) Add(addrs ...string) *AddressSet {
	for _, addr := range addrs {
		as.M[addr] = true
	}
	return as
}

// RemoveBogons MUTATES the set to remove bogons from the set.
func (as *AddressSet) RemoveBogons() *AddressSet {
	zap := []string{}
	for addr := range as.M {
		if netxlite.IsBogon(addr) {
			zap = append(zap, addr)
		}
	}
	for _, addr := range zap {
		delete(as.M, addr)
	}
	return as
}

// EndpointPort is the port for an endpoint.
type EndpointPort uint16

// ToEndpoints transforms this set of IP addresses to a list of endpoints. We will
// combine each IP address with the network and the port to construct an endpoint and
// we will also apply any additional option to each endpoint.
func (as *AddressSet) ToEndpoints(
	network EndpointNetwork, port EndpointPort, options ...EndpointOption) (v []*Endpoint) {
	for addr := range as.M {
		v = append(v, NewEndpoint(
			network,
			EndpointAddress(net.JoinHostPort(addr, strconv.Itoa(int(port)))),
			options...,
		))
	}
	return
}
