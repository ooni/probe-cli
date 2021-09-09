package mocks

// Encoder allows mocking dnsx.Encoder.
type Encoder struct {
	MockEncode func(domain string, qtype uint16, padding bool) ([]byte, error)
}

// Encode calls MockEncode.
func (e *Encoder) Encode(domain string, qtype uint16, padding bool) ([]byte, error) {
	return e.MockEncode(domain, qtype, padding)
}
