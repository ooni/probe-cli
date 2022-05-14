package mocks

import (
	"errors"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestDNSDecoder(t *testing.T) {
	t.Run("DecodeLookupHost", func(t *testing.T) {
		expected := errors.New("mocked error")
		e := &DNSDecoder{
			MockDecodeLookupHost: func(qtype uint16, reply []byte, queryID uint16) ([]string, error) {
				return nil, expected
			},
		}
		out, err := e.DecodeLookupHost(dns.TypeA, make([]byte, 17), dns.Id())
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
			MockDecodeHTTPS: func(reply []byte, queryID uint16) (*model.HTTPSSvc, error) {
				return nil, expected
			},
		}
		out, err := e.DecodeHTTPS(make([]byte, 17), dns.Id())
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})

	t.Run("DecodeReply", func(t *testing.T) {
		expected := errors.New("mocked error")
		e := &DNSDecoder{
			MockDecodeReply: func(reply []byte, queryID uint16) (*dns.Msg, error) {
				return nil, expected
			},
		}
		out, err := e.DecodeReply(make([]byte, 17), dns.Id())
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})
}
