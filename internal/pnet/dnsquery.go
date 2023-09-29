package pnet

import "github.com/ooni/probe-cli/v3/internal/model"

// DNSQuery contains information to query the DNS and to initiate
// a measurement pipeline based on DNS results.
type DNSQuery struct {
	// Domain is the MANDATORY domain to resolve.
	Domain string

	// EndpointNetwork is the endpoint network ("tcp" or "udp"). This field is
	// MANDATORY if you plan to pipe the results to endpoint functions.
	EndpointNetwork string

	// EndpointPort contains the endpoint port to use. This field is MANDATORY if
	// you plan to pipe the results of DNS lookups to endpoint functions.
	EndpointPort string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger
}

var _ Sharable = DNSQuery{}

// Sharable implements Sharable.
func (DNSQuery) Sharable() {
	// empty!
}
