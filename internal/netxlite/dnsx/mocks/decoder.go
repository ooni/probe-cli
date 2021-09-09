package mocks

// Decoder allows mocking dnsx.Decoder.
type Decoder struct {
	MockDecode func(qtype uint16, reply []byte) ([]string, error)
}

// Decode calls MockDecode.
func (e *Decoder) Decode(qtype uint16, reply []byte) ([]string, error) {
	return e.MockDecode(qtype, reply)
}
