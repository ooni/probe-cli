package mocks

// DNSEncoder allows mocking dnsx.DNSEncoder.
type DNSEncoder struct {
	MockEncode func(domain string, qtype uint16, padding bool) ([]byte, error)
}

// Encode calls MockEncode.
func (e *DNSEncoder) Encode(domain string, qtype uint16, padding bool) ([]byte, error) {
	return e.MockEncode(domain, qtype, padding)
}
