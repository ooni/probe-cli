package mocks

import (
	"bytes"
	"errors"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestDNSQuery(t *testing.T) {
	t.Run("Domain", func(t *testing.T) {
		expected := "dns.google."
		q := &DNSQuery{
			MockDomain: func() string {
				return expected
			},
		}
		if q.Domain() != expected {
			t.Fatal("invalid domain")
		}
	})

	t.Run("Type", func(t *testing.T) {
		expected := dns.TypeAAAA
		q := &DNSQuery{
			MockType: func() uint16 {
				return expected
			},
		}
		if q.Type() != expected {
			t.Fatal("invalid type")
		}
	})

	t.Run("Bytes", func(t *testing.T) {
		expected := []byte{0xde, 0xea, 0xad, 0xbe, 0xef}
		q := &DNSQuery{
			MockBytes: func() ([]byte, error) {
				return expected, nil
			},
		}
		out, err := q.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, out) {
			t.Fatal("invalid bytes")
		}
	})

	t.Run("ID", func(t *testing.T) {
		expected := dns.Id()
		q := &DNSQuery{
			MockID: func() uint16 {
				return expected
			},
		}
		if q.ID() != expected {
			t.Fatal("invalid id")
		}
	})
}

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
