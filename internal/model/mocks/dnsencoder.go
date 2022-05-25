package mocks

import "github.com/ooni/probe-cli/v3/internal/model"

// DNSQuery allocks mocking model.DNSQuery.
type DNSQuery struct {
	MockDomain func() string
	MockType   func() uint16
	MockBytes  func() ([]byte, error)
	MockID     func() uint16
}

func (q *DNSQuery) Domain() string {
	return q.MockDomain()
}

func (q *DNSQuery) Type() uint16 {
	return q.MockType()
}

func (q *DNSQuery) Bytes() ([]byte, error) {
	return q.MockBytes()
}

func (q *DNSQuery) ID() uint16 {
	return q.MockID()
}

var _ model.DNSQuery = &DNSQuery{}

// DNSEncoder allows mocking model.DNSEncoder.
type DNSEncoder struct {
	MockEncode func(domain string, qtype uint16, padding bool) model.DNSQuery
}

var _ model.DNSEncoder = &DNSEncoder{}

// Encode calls MockEncode.
func (e *DNSEncoder) Encode(domain string, qtype uint16, padding bool) model.DNSQuery {
	return e.MockEncode(domain, qtype, padding)
}
