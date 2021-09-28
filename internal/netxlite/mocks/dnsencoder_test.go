package mocks

import (
	"errors"
	"testing"

	"github.com/miekg/dns"
)

func TestDNSEncoder(t *testing.T) {
	t.Run("Encode", func(t *testing.T) {
		expected := errors.New("mocked error")
		e := &DNSEncoder{
			MockEncode: func(domain string, qtype uint16, padding bool) ([]byte, error) {
				return nil, expected
			},
		}
		out, err := e.Encode("dns.google", dns.TypeA, true)
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})
}
