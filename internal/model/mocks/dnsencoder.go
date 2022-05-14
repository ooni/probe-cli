package mocks

// DNSEncoder allows mocking dnsx.DNSEncoder.
type DNSEncoder struct {
	MockEncode func(domain string, qtype uint16, padding bool) ([]byte, uint16, error)
}

// Encode calls MockEncode.
func (e *DNSEncoder) Encode(domain string, qtype uint16, padding bool) ([]byte, uint16, error) {
	return e.MockEncode(domain, qtype, padding)
}
