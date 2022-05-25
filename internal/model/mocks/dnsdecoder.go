package mocks

import (
	"net"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSResponse allows mocking model.DNSResponse.
type DNSResponse struct {
	MockQuery            func() model.DNSQuery
	MockMessage          func() *dns.Msg
	MockBytes            func() []byte
	MockRcode            func() int
	MockDecodeHTTPS      func() (*model.HTTPSSvc, error)
	MockDecodeLookupHost func() ([]string, error)
	MockDecodeNS         func() ([]*net.NS, error)
}

var _ model.DNSResponse = &DNSResponse{}

func (r *DNSResponse) Query() model.DNSQuery {
	return r.MockQuery()
}

func (r *DNSResponse) Message() *dns.Msg {
	return r.MockMessage()
}

func (r *DNSResponse) Bytes() []byte {
	return r.MockBytes()
}

func (r *DNSResponse) Rcode() int {
	return r.MockRcode()
}

func (r *DNSResponse) DecodeHTTPS() (*model.HTTPSSvc, error) {
	return r.MockDecodeHTTPS()
}

func (r *DNSResponse) DecodeLookupHost() ([]string, error) {
	return r.MockDecodeLookupHost()
}

func (r *DNSResponse) DecodeNS() ([]*net.NS, error) {
	return r.MockDecodeNS()
}

// DNSDecoder allows mocking model.DNSDecoder.
type DNSDecoder struct {
	MockDecodeResponse func(data []byte, query model.DNSQuery) (model.DNSResponse, error)
}

var _ model.DNSDecoder = &DNSDecoder{}

func (e *DNSDecoder) DecodeResponse(data []byte, query model.DNSQuery) (model.DNSResponse, error) {
	return e.MockDecodeResponse(data, query)
}
