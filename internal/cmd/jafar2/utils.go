package main

import (
	"errors"
	"math"
	"net"
	"strconv"
)

// localRemoteAddressProvider is anything we can obtain the address of.
type localRemoteAddressProvider interface {
	// LocalAddr returns the local address
	LocalAddr() net.Addr

	// RemoteAddr returns the remote address
	RemoteAddr() net.Addr
}

var (
	// errInvalidPort indicates we were passed an invalid port
	errInvalidPort = errors.New("invalid port")

	// errInvalidIP indicates we were passed an invalid IP
	errInvalidIP = errors.New("invalid IP address")
)

// fourTuple returns the five tuple of a given net.Conn
func fourTuple(
	addressable localRemoteAddressProvider) (srcIP net.IP, srcPort uint16, dstIP net.IP, dstPort uint16, err error) {
	srcIP, srcPort, err = twoTuple(addressable.LocalAddr())
	if err != nil {
		return
	}
	dstIP, dstPort, err = twoTuple(addressable.RemoteAddr())
	return
}

// twoTuple returns the two tuple from the given net.Addr
func twoTuple(addr net.Addr) (net.IP, uint16, error) {
	maybeIP, maybePort, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil, 0, err
	}
	uport, err := strconv.Atoi(maybePort)
	if err != nil {
		return nil, 0, err
	}
	if uport < 0 || uport >= math.MaxUint16 {
		return nil, 0, errInvalidPort
	}
	ip := net.ParseIP(maybeIP)
	if ip == nil {
		return nil, 0, errInvalidIP
	}
	return ip, uint16(uport), nil
}
