package netxlite

import (
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
)

func TestDNSDecoder(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
		t.Run("UnpackError", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			data, err := d.DecodeLookupHost(dns.TypeA, nil)
			if err == nil {
				t.Fatal("expected an error here")
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("NXDOMAIN", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			data, err := d.DecodeLookupHost(
				dns.TypeA, dnsGenReplyWithError(t, dns.TypeA, dns.RcodeNameError))
			if err == nil || !strings.HasSuffix(err.Error(), "no such host") {
				t.Fatal("not the error we expected", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("Refused", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			data, err := d.DecodeLookupHost(
				dns.TypeA, dnsGenReplyWithError(t, dns.TypeA, dns.RcodeRefused))
			if !errors.Is(err, ErrOODNSRefused) {
				t.Fatal("not the error we expected", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("no address", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			data, err := d.DecodeLookupHost(dns.TypeA, dnsGenLookupHostReplySuccess(t, dns.TypeA))
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("not the error we expected", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("decode A", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			data, err := d.DecodeLookupHost(
				dns.TypeA, dnsGenLookupHostReplySuccess(t, dns.TypeA, "1.1.1.1", "8.8.8.8"))
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
		})

		t.Run("decode AAAA", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			data, err := d.DecodeLookupHost(
				dns.TypeAAAA, dnsGenLookupHostReplySuccess(t, dns.TypeAAAA, "::1", "fe80::1"))
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
		})

		t.Run("unexpected A reply", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			data, err := d.DecodeLookupHost(
				dns.TypeA, dnsGenLookupHostReplySuccess(t, dns.TypeAAAA, "::1", "fe80::1"))
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("not the error we expected", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("unexpected AAAA reply", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			data, err := d.DecodeLookupHost(
				dns.TypeAAAA, dnsGenLookupHostReplySuccess(t, dns.TypeA, "1.1.1.1", "8.8.4.4."))
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("not the error we expected", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})
	})

	t.Run("parseReply", func(t *testing.T) {
		d := &DNSDecoderMiekg{}
		msg := &dns.Msg{}
		msg.Rcode = dns.RcodeFormatError // an rcode we don't handle
		data, err := msg.Pack()
		if err != nil {
			t.Fatal(err)
		}
		reply, err := d.parseReply(data)
		if !errors.Is(err, ErrOODNSMisbehaving) { // catch all error
			t.Fatal("not the error we expected", err)
		}
		if reply != nil {
			t.Fatal("expected nil reply")
		}
	})

	t.Run("DecodeHTTPS", func(t *testing.T) {
		t.Run("with nil data", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			reply, err := d.DecodeHTTPS(nil)
			if err == nil || err.Error() != "dns: overflow unpacking uint16" {
				t.Fatal("not the error we expected", err)
			}
			if reply != nil {
				t.Fatal("expected nil reply")
			}
		})

		t.Run("with empty answer", func(t *testing.T) {
			data := dnsGenHTTPSReplySuccess(t, nil, nil, nil)
			d := &DNSDecoderMiekg{}
			reply, err := d.DecodeHTTPS(data)
			if !errors.Is(err, ErrOODNSNoAnswer) {
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
			data := dnsGenHTTPSReplySuccess(t, alpn, v4, v6)
			d := &DNSDecoderMiekg{}
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
	})
}

// dnsGenReplyWithError generates a DNS reply for the given
// query type (e.g., dns.TypeA) using code as the Rcode.
func dnsGenReplyWithError(t *testing.T, qtype uint16, code int) []byte {
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
	reply.SetRcode(query, code)
	data, err := reply.Pack()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// dnsGenLookupHostReplySuccess generates a successful DNS reply for the given
// qtype (e.g., dns.TypeA) containing the given ips... in the answer.
func dnsGenLookupHostReplySuccess(t *testing.T, qtype uint16, ips ...string) []byte {
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

// dnsGenHTTPSReplySuccess generates a successful HTTPS response containing
// the given (possibly nil) alpns, ipv4s, and ipv6s.
func dnsGenHTTPSReplySuccess(t *testing.T, alpns, ipv4s, ipv6s []string) []byte {
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
				Name:   dns.Fqdn("x.org"),
				Rrtype: dns.TypeHTTPS,
				Class:  dns.ClassINET,
				Ttl:    100,
			},
			Target: dns.Fqdn("x.org"),
			Value:  []dns.SVCBKeyValue{},
		},
	}
	reply.Answer = append(reply.Answer, answer)
	if len(alpns) > 0 {
		answer.Value = append(answer.Value, &dns.SVCBAlpn{Alpn: alpns})
	}
	if len(ipv4s) > 0 {
		var addrs []net.IP
		for _, addr := range ipv4s {
			addrs = append(addrs, net.ParseIP(addr))
		}
		answer.Value = append(answer.Value, &dns.SVCBIPv4Hint{Hint: addrs})
	}
	if len(ipv6s) > 0 {
		var addrs []net.IP
		for _, addr := range ipv6s {
			addrs = append(addrs, net.ParseIP(addr))
		}
		answer.Value = append(answer.Value, &dns.SVCBIPv6Hint{Hint: addrs})
	}
	data, err := reply.Pack()
	if err != nil {
		t.Fatal(err)
	}
	return data
}
