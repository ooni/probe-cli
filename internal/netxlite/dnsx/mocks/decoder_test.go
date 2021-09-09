package mocks

import (
	"errors"
	"testing"

	"github.com/miekg/dns"
)

func TestDecoder(t *testing.T) {
	t.Run("Decode", func(t *testing.T) {
		expected := errors.New("mocked error")
		e := &Decoder{
			MockDecode: func(qtype uint16, reply []byte) ([]string, error) {
				return nil, expected
			},
		}
		out, err := e.Decode(dns.TypeA, make([]byte, 17))
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})
}
