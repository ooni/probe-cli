package mocks

//
// Mocks for model.DNSDecoder
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSDecoder allows mocking model.DNSDecoder.
type DNSDecoder struct {
	MockDecodeResponse func(data []byte, query model.DNSQuery) (model.DNSResponse, error)
}

var _ model.DNSDecoder = &DNSDecoder{}

func (e *DNSDecoder) DecodeResponse(data []byte, query model.DNSQuery) (model.DNSResponse, error) {
	return e.MockDecodeResponse(data, query)
}
