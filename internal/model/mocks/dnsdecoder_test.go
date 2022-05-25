package mocks

import (
	"bytes"
	"errors"
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestDNSResponse(t *testing.T) {
	t.Run("Query", func(t *testing.T) {
		qid := dns.Id()
		query := &DNSQuery{
			MockID: func() uint16 {
				return qid
			},
		}
		resp := &DNSResponse{
			MockQuery: func() model.DNSQuery {
				return query
			},
		}
		out := resp.Query()
		if out.ID() != query.ID() {
			t.Fatal("invalid query")
		}
	})

	t.Run("Message", func(t *testing.T) {
		msg := &dns.Msg{}
		msg.Id = dns.Id()
		resp := &DNSResponse{
			MockMessage: func() *dns.Msg {
				return msg
			},
		}
		out := resp.Message()
		if out.Id != msg.Id {
			t.Fatal("invalid message")
		}
	})

	t.Run("Bytes", func(t *testing.T) {
		expected := []byte{0xde, 0xea, 0xad, 0xbe, 0xef}
		resp := &DNSResponse{
			MockBytes: func() []byte {
				return expected
			},
		}
		out := resp.Bytes()
		if !bytes.Equal(expected, out) {
			t.Fatal("invalid bytes")
		}
	})

	t.Run("Rcode", func(t *testing.T) {
		expected := dns.RcodeBadAlg
		resp := &DNSResponse{
			MockRcode: func() int {
				return expected
			},
		}
		out := resp.Rcode()
		if out != expected {
			t.Fatal("invalid rcode")
		}
	})

	t.Run("DecodeLookupHost", func(t *testing.T) {
		expected := errors.New("mocked error")
		r := &DNSResponse{
			MockDecodeLookupHost: func() ([]string, error) {
				return nil, expected
			},
		}
		out, err := r.DecodeLookupHost()
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})

	t.Run("DecodeHTTPS", func(t *testing.T) {
		expected := errors.New("mocked error")
		r := &DNSResponse{
			MockDecodeHTTPS: func() (*model.HTTPSSvc, error) {
				return nil, expected
			},
		}
		out, err := r.DecodeHTTPS()
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})

	t.Run("DecodeNS", func(t *testing.T) {
		expected := errors.New("mocked error")
		r := &DNSResponse{
			MockDecodeNS: func() ([]*net.NS, error) {
				return nil, expected
			},
		}
		out, err := r.DecodeNS()
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})
}

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
