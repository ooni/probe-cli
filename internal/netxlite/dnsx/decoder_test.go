package dnsx

import (
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
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
		t.Fatal("not the error we expected", err)
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderRefusedError(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.DecodeLookupHost(dns.TypeA, genReplyError(t, dns.RcodeRefused))
	if !errors.Is(err, errorsx.ErrOODNSRefused) {
		t.Fatal("not the error we expected", err)
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderNoAddress(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.DecodeLookupHost(dns.TypeA, genReplySuccess(t, dns.TypeA))
	if !errors.Is(err, errorsx.ErrOODNSNoAnswer) {
		t.Fatal("not the error we expected", err)
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
	if !errors.Is(err, errorsx.ErrOODNSNoAnswer) {
		t.Fatal("not the error we expected", err)
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderUnexpectedAAAAReply(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.DecodeLookupHost(
		dns.TypeAAAA, genReplySuccess(t, dns.TypeA, "1.1.1.1", "8.8.4.4."))
	if !errors.Is(err, errorsx.ErrOODNSNoAnswer) {
		t.Fatal("not the error we expected", err)
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

func TestParseReply(t *testing.T) {
	d := &MiekgDecoder{}
	msg := &dns.Msg{}
	msg.Rcode = dns.RcodeFormatError // an rcode we don't handle
	data, err := msg.Pack()
	if err != nil {
		t.Fatal(err)
	}
	reply, err := d.parseReply(data)
	if !errors.Is(err, errorsx.ErrOODNSMisbehaving) { // catch all error
		t.Fatal("not the error we expected", err)
	}
	if reply != nil {
		t.Fatal("expected nil reply")
	}
}

func genReplyHTTPS(t *testing.T, alpns, ipv4, ipv6 []string) []byte {
	question := dns.Question{
		Name:   dns.Fqdn("x.org"),
		Qtype:  dns.TypeHTTPS,
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
	answer := &dns.HTTPS{
		SVCB: dns.SVCB{
			Hdr: dns.RR_Header{
				Name:     dns.Fqdn("x.org"),
				Rrtype:   dns.TypeHTTPS,
				Class:    dns.ClassINET,
				Ttl:      100,
				Rdlength: 0,
			},
			Priority: 5,
			Target:   dns.Fqdn("x.org"),
			Value:    []dns.SVCBKeyValue{},
		},
	}
	reply.Answer = append(reply.Answer, answer)
	if len(alpns) > 0 {
		answer.Value = append(answer.Value, &dns.SVCBAlpn{
			Alpn: alpns,
		})
		answer.Hdr.Rdlength++
	}
	if len(ipv4) > 0 {
		var addrs []net.IP
		for _, addr := range ipv4 {
			addrs = append(addrs, net.ParseIP(addr))
		}
		answer.Value = append(answer.Value, &dns.SVCBIPv4Hint{
			Hint: addrs,
		})
		answer.Hdr.Rdlength++
	}
	if len(ipv6) > 0 {
		var addrs []net.IP
		for _, addr := range ipv6 {
			addrs = append(addrs, net.ParseIP(addr))
		}
		answer.Value = append(answer.Value, &dns.SVCBIPv6Hint{
			Hint: addrs,
		})
		answer.Hdr.Rdlength++
	}
	data, err := reply.Pack()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDecodeHTTPS(t *testing.T) {
	t.Run("with nil data", func(t *testing.T) {
		d := &MiekgDecoder{}
		reply, err := d.DecodeHTTPS(nil)
		if err == nil || err.Error() != "dns: overflow unpacking uint16" {
			t.Fatal("not the error we expected", err)
		}
		if reply != nil {
			t.Fatal("expected nil reply")
		}
	})

	t.Run("with empty answer", func(t *testing.T) {
		data := genReplyHTTPS(t, nil, nil, nil)
		d := &MiekgDecoder{}
		reply, err := d.DecodeHTTPS(data)
		if !errors.Is(err, errorsx.ErrOODNSNoAnswer) {
			t.Fatal("unexpected err", err)
		}
		if reply != nil {
			t.Fatal("expected nil reply")
		}
	})

	t.Run("with full answer", func(t *testing.T) {
		alpn := []string{"h3"}
		v4 := []string{"1.1.1.1"}
		v6 := []string{"::1"}
		data := genReplyHTTPS(t, alpn, v4, v6)
		d := &MiekgDecoder{}
		reply, err := d.DecodeHTTPS(data)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(alpn, reply.ALPN); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(v4, reply.IPv4); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(v6, reply.IPv6); diff != "" {
			t.Fatal(diff)
		}
	})
}
