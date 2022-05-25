package mocks

import (
	"errors"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestDNSEncoder(t *testing.T) {
	t.Run("Encode", func(t *testing.T) {
		expected := errors.New("mocked error")
		queryID := dns.Id()
		e := &DNSEncoder{
			MockEncode: func(domain string, qtype uint16, padding bool) model.DNSQuery {
				return &DNSQuery{
					MockDomain: func() string {
						return dns.Fqdn(domain) // do what an implementation MUST do
					},
					MockType: func() uint16 {
						return qtype
					},
					MockBytes: func() ([]byte, error) {
						return nil, expected
					},
					MockID: func() uint16 {
						return queryID
					},
				}
			},
		}
		query := e.Encode("dns.google", dns.TypeA, true)
		if query.Domain() != "dns.google." {
			t.Fatal("invalid domain")
		}
		if query.Type() != dns.TypeA {
			t.Fatal("invalid type")
		}
		out, err := query.Bytes()
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
		if query.ID() != queryID {
			t.Fatal("unexpected queryID")
		}
	})
}
