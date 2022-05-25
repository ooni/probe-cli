package mocks

import (
	"bytes"
	"testing"

	"github.com/miekg/dns"
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
