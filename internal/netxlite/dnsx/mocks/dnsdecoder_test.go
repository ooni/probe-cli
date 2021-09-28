package mocks

import (
	"errors"
	"testing"

	"github.com/miekg/dns"
)

func TestDNSDecoder(t *testing.T) {
	t.Run("DecodeLookupHost", func(t *testing.T) {
		expected := errors.New("mocked error")
		e := &DNSDecoder{
			MockDecodeLookupHost: func(qtype uint16, reply []byte) ([]string, error) {
				return nil, expected
			},
		}
		out, err := e.DecodeLookupHost(dns.TypeA, make([]byte, 17))
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})

	t.Run("DecodeHTTPS", func(t *testing.T) {
		expected := errors.New("mocked error")
		e := &DNSDecoder{
			MockDecodeHTTPS: func(reply []byte) (*HTTPSSvc, error) {
				return nil, expected
			},
		}
		out, err := e.DecodeHTTPS(make([]byte, 17))
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})
}
