package main

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/pborman/getopt/v2"
)

var (
	// regexpChain identifies valid iptables chains
	regexpChain = regexp.MustCompile("^[A-Z0-9_]{1,16}$")

	// regexpDevice identifies valid device names
	regexpDevice = regexp.MustCompile("^[a-zA-z0-9]{1,16}$")

	// regexpNetns identifies valid netns names
	regexpNetns = regexp.MustCompile("^[a-zA-z0-9]{1,16}$")

	// regexpNetmask identifies valid netmasks
	regexpNetmask = regexp.MustCompile("^[1-9][0-9]?$")

	// regexpUserGroup identifiers a valid user or group name
	regexpUserGroup = regexp.MustCompile("^[a-zA-Z0-9]{1,64}$")
)

// OptEndpoint is an endpoint descriptor.
type OptEndpoint struct {
	// Address is the endpoint address (e.g., "1.1.1.1")
	Address string

	// Port is the endpoint port (e.g., "443").
	Port string

	// Network is the endpoint network (e.g., "tcp")
	Network string
}

// ErrInvalidEndpointString indicates that the endpoint string passed
// from command line is not a correct endpoint representation.
var ErrInvalidEndpointString = errors.New("invalid endpoint string")

// Set implements getopt.Value.Set.
func (op *OptEndpoint) Set(value string, opt getopt.Option) error {
	err := fmt.Errorf("%w: %s", ErrInvalidEndpointString, value)
	for _, suffix := range []string{"/tcp", "/udp"} {
		idx := strings.Index(value, suffix)
		if idx < 0 {
			continue
		}
		op.Network = suffix[1:]
		value = value[:idx]
		addr, port, err := net.SplitHostPort(value)
		if err != nil {
			return err
		}
		op.Address = addr
		op.Port = port
		return nil
	}
	return err
}

// String implements getopt.Value.String.
func (o *OptEndpoint) String() string {
	return fmt.Sprintf(
		"%s/%s", net.JoinHostPort(o.Address, o.Port), o.Network)
}

// OptEndpointsList is a list of endpoints.
type OptEndpointsList struct {
	// Endpoints contains all the endpoints.
	Endpoints []OptEndpoint
}

// Set implements getopt.Value.Set.
func (o *OptEndpointsList) Set(value string, opt getopt.Option) error {
	epnt := &OptEndpoint{}
	if err := epnt.Set(value, opt); err != nil {
		return err
	}
	o.Endpoints = append(o.Endpoints, *epnt)
	return nil
}

// String implements getopt.Value.String.
func (o *OptEndpointsList) String() string {
	var repr []string
	for _, epnt := range o.Endpoints {
		repr = append(repr, epnt.String())
	}
	return strings.Join(repr, ",")
}

// OptIptablesChain is an iptables chain name.
type OptIptablesChain string

// Set implements getopt.Value.Set.
func (o *OptIptablesChain) Set(value string, opt getopt.Option) error {
	if !regexpChain.MatchString(value) {
		return errors.New("invalid name for iptables chain")
	}
	*o = OptIptablesChain(value)
	return nil
}

// String implements getopt.Value.String.
func (o *OptIptablesChain) String() string {
	return string(*o)
}

// OptNetns is a network namespace name.
type OptNetns string

// Set implements getopt.Value.Set.
func (o *OptNetns) Set(value string, opt getopt.Option) error {
	if !regexpNetns.MatchString(value) {
		return errors.New("invalid name for network namespace")
	}
	*o = OptNetns(value)
	return nil
}

// String implements getopt.Value.String.
func (o *OptNetns) String() string {
	return string(*o)
}

// OptDevice is the name of a network device.
type OptDevice string

// Set implements getopt.Value.Set.
func (o *OptDevice) Set(value string, opt getopt.Option) error {
	if !regexpDevice.MatchString(value) {
		return errors.New("invalid name for network device")
	}
	*o = OptDevice(value)
	return nil
}

// String implements getopt.Value.String.
func (o *OptDevice) String() string {
	return string(*o)
}

// OptIPAddress is an IP address.
type OptIPAddress string

// Set implements getopt.Value.Set.
func (o *OptIPAddress) Set(value string, opt getopt.Option) error {
	if net.ParseIP(value) == nil {
		return errors.New("invalid value for IP address")
	}
	*o = OptIPAddress(value)
	return nil
}

// String implements getopt.Value.String.
func (o *OptIPAddress) String() string {
	return string(*o)
}

// OptNetmask is a netmask.
type OptNetmask string

// Set implements getopt.Value.Set.
func (o *OptNetmask) Set(value string, opt getopt.Option) error {
	if !regexpNetmask.MatchString(value) {
		return errors.New("invalid value for netmask")
	}
	*o = OptNetmask(value)
	return nil
}

// String implements getopt.Value.String.
func (o *OptNetmask) String() string {
	return string(*o)
}

// OptUserGroup is the name of a user or group.
type OptUserGroup string

// Set implements getopt.Value.Set.
func (o *OptUserGroup) Set(value string, opt getopt.Option) error {
	if !regexpUserGroup.MatchString(value) {
		return errors.New("invalid value for netmask")
	}
	*o = OptUserGroup(value)
	return nil
}

// String implements getopt.Value.String.
func (o *OptUserGroup) String() string {
	return string(*o)
}
