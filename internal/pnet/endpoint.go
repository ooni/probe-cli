package pnet

import "github.com/ooni/probe-cli/v3/internal/model"

// Endpoint contains information to establish a TCP or QUIC connection and
// to initiate a measurement pipeline using such a connection.
type Endpoint struct {
	// Domain is the OPTIONAL domain from which we resolved IPAddress.

	// IPAddress is the MANDATORY IP address.
	IPAddress string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// Network is the MANDATORY network ("tcp" or "udp").
	Network string

	// Port is the MANDATORY port.
	Port string
}

var _ Sharable = Endpoint{}

// Sharable implements Sharable.
func (Endpoint) Sharable() {
	// empty!
}
