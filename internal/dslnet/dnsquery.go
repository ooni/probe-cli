package dslnet

import "github.com/ooni/probe-cli/v3/internal/dslmodel"

// DNSQuery contains information to query the DNS and to initiate
// a measurement pipeline based on DNS results.
type DNSQuery struct {
	// Domain is the MANDATORY domain to resolve.
	Domain string

	// EndpointTemplate is the template for creating [Endpoint]. This field
	// is MANDATORY if you plan on using endpoint functions.
	EndpointTemplate EndpointTemplate

	// Tags contains OPTIONAL tags to tag OONI observations.
	Tags []string
}

var _ dslmodel.Sharable = DNSQuery{}

// Sharable implements dslmodel.Sharable.
func (DNSQuery) Sharable() {
	// nothing
}
