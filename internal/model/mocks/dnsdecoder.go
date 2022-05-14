package mocks

import (
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSDecoder allows mocking dnsx.DNSDecoder.
type DNSDecoder struct {
	MockDecodeLookupHost func(qtype uint16, reply []byte, queryID uint16) ([]string, error)

	MockDecodeHTTPS func(reply []byte, queryID uint16) (*model.HTTPSSvc, error)

	MockDecodeReply func(reply []byte, queryID uint16) (*dns.Msg, error)
}

// DecodeLookupHost calls MockDecodeLookupHost.
func (e *DNSDecoder) DecodeLookupHost(qtype uint16, reply []byte, queryID uint16) ([]string, error) {
	return e.MockDecodeLookupHost(qtype, reply, queryID)
}

// DecodeHTTPS calls MockDecodeHTTPS.
func (e *DNSDecoder) DecodeHTTPS(reply []byte, queryID uint16) (*model.HTTPSSvc, error) {
	return e.MockDecodeHTTPS(reply, queryID)
}

// DecodeReply calls MockDecodeReply.
func (e *DNSDecoder) DecodeReply(reply []byte, queryID uint16) (*dns.Msg, error) {
	return e.MockDecodeReply(reply, queryID)
}
