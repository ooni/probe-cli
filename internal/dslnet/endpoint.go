package dslnet

import (
	"fmt"
	"net"

	"github.com/ooni/probe-cli/v3/internal/dslmodel"
)

// Endpoint contains information to establish a TCP or QUIC connection and
// to initiate a measurement pipeline using such a connection.
type Endpoint struct {
	// Domain is the OPTIONAL domain from which we resolved IPAddress.
	Domain string

	// IPAddress is the MANDATORY IP address.
	IPAddress string

	// Network is the MANDATORY network ("tcp" or "udp").
	Network string

	// Port is the MANDATORY port.
	Port string

	// Tags contains OPTIONAL tags to tag OONI observations.
	Tags []string
}

var _ dslmodel.Sharable = Endpoint{}

// Sharable implements dslmodel.Sharable.
func (e Endpoint) Sharable() {
	// nothing
}

var _ dslmodel.Deduplicable = Endpoint{}

// DedupKey implements dslmodel.Deduplicable.
func (e Endpoint) DedupKey() string {
	return fmt.Sprintf("%s/%s", net.JoinHostPort(e.IPAddress, e.Port), e.Network)
}

// EndpointTemplate is the template used for creating an [Endpoint].
type EndpointTemplate struct {
	// Network is the MANDATORY network ("tcp" or "udp").
	Network string

	// Port is the MANDATORY port.
	Port string

	// Tags contains OPTIONAL tags to tag OONI observations.
	Tags []string
}

var _ dslmodel.Sharable = EndpointTemplate{}

// Sharable implements dslmodel.Sharable.
func (EndpointTemplate) Sharable() {
	// nothing
}

// NewEndpoint creates a new [Endpoint].
func NewEndpoint(template EndpointTemplate, domain, ipAddr string) Endpoint {
	return Endpoint{
		Domain:    domain,
		IPAddress: ipAddr,
		Network:   template.Network,
		Port:      template.Port,
		Tags:      append([]string{}, template.Tags...),
	}
}
