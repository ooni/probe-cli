package mocks

//
// Mocks for model.DNSResponse
//

import (
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSResponse allows mocking model.DNSResponse.
type DNSResponse struct {
	MockQuery            func() model.DNSQuery
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
