package mocks

import "github.com/ooni/probe-cli/v3/internal/model"

// DNSDecoder allows mocking dnsx.DNSDecoder.
type DNSDecoder struct {
	MockDecodeLookupHost func(qtype uint16, reply []byte) ([]string, error)

	MockDecodeHTTPS func(reply []byte) (*model.HTTPSSvc, error)
}

// DecodeLookupHost calls MockDecodeLookupHost.
func (e *DNSDecoder) DecodeLookupHost(qtype uint16, reply []byte) ([]string, error) {
	return e.MockDecodeLookupHost(qtype, reply)
}

// DecodeHTTPS calls MockDecodeHTTPS.
func (e *DNSDecoder) DecodeHTTPS(reply []byte) (*model.HTTPSSvc, error) {
	return e.MockDecodeHTTPS(reply)
}
