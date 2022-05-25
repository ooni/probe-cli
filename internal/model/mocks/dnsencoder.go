package mocks

//
// Mocks for model.DNSEncoder.
//

import "github.com/ooni/probe-cli/v3/internal/model"

// DNSEncoder allows mocking model.DNSEncoder.
type DNSEncoder struct {
	MockEncode func(domain string, qtype uint16, padding bool) model.DNSQuery
}

var _ model.DNSEncoder = &DNSEncoder{}

// Encode calls MockEncode.
func (e *DNSEncoder) Encode(domain string, qtype uint16, padding bool) model.DNSQuery {
	return e.MockEncode(domain, qtype, padding)
}
