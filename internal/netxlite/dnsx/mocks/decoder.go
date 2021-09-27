package mocks

import "github.com/ooni/probe-cli/v3/internal/netxlite/dnsx/model"

// HTTPSSvc is the result of HTTPS queries.
type HTTPSSvc = model.HTTPSSvc

// Decoder allows mocking dnsx.Decoder.
type Decoder struct {
	MockDecodeLookupHost func(qtype uint16, reply []byte) ([]string, error)

	MockDecodeHTTPS func(reply []byte) (*HTTPSSvc, error)
}

// DecodeLookupHost calls MockDecodeLookupHost.
func (e *Decoder) DecodeLookupHost(qtype uint16, reply []byte) ([]string, error) {
	return e.MockDecodeLookupHost(qtype, reply)
}

// DecodeHTTPS calls MockDecodeHTTPS.
func (e *Decoder) DecodeHTTPS(reply []byte) (*HTTPSSvc, error) {
	return e.MockDecodeHTTPS(reply)
}
