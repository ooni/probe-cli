package mocks

import (
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestDNSDecoder(t *testing.T) {
	t.Run("DecodeResponse", func(t *testing.T) {
		expected := errors.New("mocked error")
		e := &DNSDecoder{
			MockDecodeResponse: func(reply []byte, query model.DNSQuery) (model.DNSResponse, error) {
				return nil, expected
			},
		}
		out, err := e.DecodeResponse(make([]byte, 17), &DNSQuery{})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})
}
