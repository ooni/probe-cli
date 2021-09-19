package dnsx

import (
	"net"
	"strings"
	"testing"

	"github.com/miekg/dns"
)

func TestDecoderUnpackError(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.DecodeLookupHost(dns.TypeA, nil)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderNXDOMAIN(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.DecodeLookupHost(dns.TypeA, genReplyError(t, dns.RcodeNameError))
	if err == nil || !strings.HasSuffix(err.Error(), "no such host") {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderOtherError(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.DecodeLookupHost(dns.TypeA, genReplyError(t, dns.RcodeRefused))
	if err == nil || !strings.HasSuffix(err.Error(), "query failed") {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderNoAddress(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.DecodeLookupHost(dns.TypeA, genReplySuccess(t, dns.TypeA))
	if err == nil || !strings.HasSuffix(err.Error(), "no response returned") {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderDecodeA(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.DecodeLookupHost(
		dns.TypeA, genReplySuccess(t, dns.TypeA, "1.1.1.1", "8.8.8.8"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 {
		t.Fatal("expected two entries here")
	}
	if data[0] != "1.1.1.1" {
		t.Fatal("invalid first IPv4 entry")
	}
	if data[1] != "8.8.8.8" {
		t.Fatal("invalid second IPv4 entry")
	}
}

func TestDecoderDecodeAAAA(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.DecodeLookupHost(
		dns.TypeAAAA, genReplySuccess(t, dns.TypeAAAA, "::1", "fe80::1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 {
		t.Fatal("expected two entries here")
	}
	if data[0] != "::1" {
		t.Fatal("invalid first IPv6 entry")
	}
	if data[1] != "fe80::1" {
		t.Fatal("invalid second IPv6 entry")
	}
}

func TestDecoderUnexpectedAReply(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.DecodeLookupHost(
		dns.TypeA, genReplySuccess(t, dns.TypeAAAA, "::1", "fe80::1"))
	if err == nil || !strings.HasSuffix(err.Error(), "no response returned") {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderUnexpectedAAAAReply(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.DecodeLookupHost(
		dns.TypeAAAA, genReplySuccess(t, dns.TypeA, "1.1.1.1", "8.8.4.4."))
	if err == nil || !strings.HasSuffix(err.Error(), "no response returned") {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func genReplyError(t *testing.T, code int) []byte {
	question := dns.Question{
		Name:   dns.Fqdn("x.org"),
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}
	query := new(dns.Msg)
	query.Id = dns.Id()
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.Question[0] = question
	reply := new(dns.Msg)
	reply.Compress = true
	reply.MsgHdr.RecursionAvailable = true
	reply.SetRcode(query, code)
	data, err := reply.Pack()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func genReplySuccess(t *testing.T, qtype uint16, ips ...string) []byte {
	question := dns.Question{
		Name:   dns.Fqdn("x.org"),
		Qtype:  qtype,
		Qclass: dns.ClassINET,
	}
	query := new(dns.Msg)
	query.Id = dns.Id()
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.Question[0] = question
	reply := new(dns.Msg)
	reply.Compress = true
	reply.MsgHdr.RecursionAvailable = true
	reply.SetReply(query)
	for _, ip := range ips {
		switch qtype {
		case dns.TypeA:
			reply.Answer = append(reply.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn("x.org"),
					Rrtype: qtype,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				A: net.ParseIP(ip),
			})
		case dns.TypeAAAA:
			reply.Answer = append(reply.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn("x.org"),
					Rrtype: qtype,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				AAAA: net.ParseIP(ip),
			})
		}
	}
	data, err := reply.Pack()
	if err != nil {
		t.Fatal(err)
	}
	return data
}
