package netxlite

import (
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestDNSDecoder(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
		t.Run("UnpackError", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			data, err := d.DecodeLookupHost(dns.TypeA, nil, 0)
			if err == nil || err.Error() != "dns: overflow unpacking uint16" {
				t.Fatal("unexpected error", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("with bytes containing a query", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			rawQuery := dnsGenQuery(dns.TypeA, queryID)
			addrs, err := d.DecodeLookupHost(dns.TypeA, rawQuery, queryID)
			if !errors.Is(err, ErrDNSIsQuery) {
				t.Fatal("unexpected err", err)
			}
			if len(addrs) > 0 {
				t.Fatal("expected no addrs")
			}
		})

		t.Run("wrong query ID", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			const (
				queryID     = 17
				unrelatedID = 14
			)
			reply := dnsGenLookupHostReplySuccess(dnsGenQuery(dns.TypeA, queryID))
			data, err := d.DecodeLookupHost(dns.TypeA, reply, unrelatedID)
			if !errors.Is(err, ErrDNSReplyWithWrongQueryID) {
				t.Fatal("unexpected error", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("NXDOMAIN", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			data, err := d.DecodeLookupHost(dns.TypeA, dnsGenReplyWithError(
				dnsGenQuery(dns.TypeA, queryID), dns.RcodeNameError), queryID)
			if err == nil || !strings.HasSuffix(err.Error(), "no such host") {
				t.Fatal("not the error we expected", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("Refused", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			data, err := d.DecodeLookupHost(dns.TypeA, dnsGenReplyWithError(
				dnsGenQuery(dns.TypeA, queryID), dns.RcodeRefused), queryID)
			if !errors.Is(err, ErrOODNSRefused) {
				t.Fatal("not the error we expected", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("Servfail", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			data, err := d.DecodeLookupHost(dns.TypeA, dnsGenReplyWithError(
				dnsGenQuery(dns.TypeA, queryID), dns.RcodeServerFailure), queryID)
			if !errors.Is(err, ErrOODNSServfail) {
				t.Fatal("not the error we expected", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("no address", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			data, err := d.DecodeLookupHost(dns.TypeA, dnsGenLookupHostReplySuccess(
				dnsGenQuery(dns.TypeA, queryID)), queryID)
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("not the error we expected", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("decode A", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			data, err := d.DecodeLookupHost(dns.TypeA, dnsGenLookupHostReplySuccess(
				dnsGenQuery(dns.TypeA, queryID), "1.1.1.1", "8.8.8.8"), queryID)
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
			queryID := dns.Id()
			data, err := d.DecodeLookupHost(dns.TypeAAAA, dnsGenLookupHostReplySuccess(
				dnsGenQuery(dns.TypeAAAA, queryID), "::1", "fe80::1"), queryID)
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
			queryID := dns.Id()
			data, err := d.DecodeLookupHost(dns.TypeA, dnsGenLookupHostReplySuccess(
				dnsGenQuery(dns.TypeAAAA, queryID), "::1", "fe80::1"), queryID)
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("not the error we expected", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("unexpected AAAA reply", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			data, err := d.DecodeLookupHost(dns.TypeAAAA, dnsGenLookupHostReplySuccess(
				dnsGenQuery(dns.TypeA, queryID), "1.1.1.1", "8.8.4.4"), queryID)
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("not the error we expected", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})
	})

	t.Run("decodeSuccessfulReply", func(t *testing.T) {
		d := &DNSDecoderMiekg{}
		msg := &dns.Msg{}
		msg.Rcode = dns.RcodeFormatError // an rcode we don't handle
		msg.Response = true
		data, err := msg.Pack()
		if err != nil {
			t.Fatal(err)
		}
		reply, err := d.decodeSuccessfulReply(data, 0)
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
			reply, err := d.DecodeHTTPS(nil, 0)
			if err == nil || err.Error() != "dns: overflow unpacking uint16" {
				t.Fatal("not the error we expected", err)
			}
			if reply != nil {
				t.Fatal("expected nil reply")
			}
		})

		t.Run("with bytes containing a query", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			rawQuery := dnsGenQuery(dns.TypeHTTPS, queryID)
			https, err := d.DecodeHTTPS(rawQuery, queryID)
			if !errors.Is(err, ErrDNSIsQuery) {
				t.Fatal("unexpected err", err)
			}
			if https != nil {
				t.Fatal("expected nil https")
			}
		})

		t.Run("wrong query ID", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			const (
				queryID     = 17
				unrelatedID = 14
			)
			reply := dnsGenHTTPSReplySuccess(dnsGenQuery(dns.TypeHTTPS, queryID), nil, nil, nil)
			data, err := d.DecodeHTTPS(reply, unrelatedID)
			if !errors.Is(err, ErrDNSReplyWithWrongQueryID) {
				t.Fatal("unexpected error", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("with empty answer", func(t *testing.T) {
			queryID := dns.Id()
			data := dnsGenHTTPSReplySuccess(
				dnsGenQuery(dns.TypeHTTPS, queryID), nil, nil, nil)
			d := &DNSDecoderMiekg{}
			reply, err := d.DecodeHTTPS(data, queryID)
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("unexpected err", err)
			}
			if reply != nil {
				t.Fatal("expected nil reply")
			}
		})

		t.Run("with full answer", func(t *testing.T) {
			queryID := dns.Id()
			alpn := []string{"h3"}
			v4 := []string{"1.1.1.1"}
			v6 := []string{"::1"}
			data := dnsGenHTTPSReplySuccess(
				dnsGenQuery(dns.TypeHTTPS, queryID), alpn, v4, v6)
			d := &DNSDecoderMiekg{}
			reply, err := d.DecodeHTTPS(data, queryID)
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

	t.Run("DecodeNS", func(t *testing.T) {
		t.Run("with nil data", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			reply, err := d.DecodeNS(nil, 0)
			if err == nil || err.Error() != "dns: overflow unpacking uint16" {
				t.Fatal("not the error we expected", err)
			}
			if reply != nil {
				t.Fatal("expected nil reply")
			}
		})

		t.Run("with bytes containing a query", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			rawQuery := dnsGenQuery(dns.TypeNS, queryID)
			ns, err := d.DecodeNS(rawQuery, queryID)
			if !errors.Is(err, ErrDNSIsQuery) {
				t.Fatal("unexpected err", err)
			}
			if len(ns) > 0 {
				t.Fatal("expected no result")
			}
		})

		t.Run("wrong query ID", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			const (
				queryID     = 17
				unrelatedID = 14
			)
			reply := dnsGenNSReplySuccess(dnsGenQuery(dns.TypeNS, queryID))
			data, err := d.DecodeNS(reply, unrelatedID)
			if !errors.Is(err, ErrDNSReplyWithWrongQueryID) {
				t.Fatal("unexpected error", err)
			}
			if data != nil {
				t.Fatal("expected nil data here")
			}
		})

		t.Run("with empty answer", func(t *testing.T) {
			queryID := dns.Id()
			data := dnsGenNSReplySuccess(dnsGenQuery(dns.TypeNS, queryID))
			d := &DNSDecoderMiekg{}
			reply, err := d.DecodeNS(data, queryID)
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("unexpected err", err)
			}
			if reply != nil {
				t.Fatal("expected nil reply")
			}
		})

		t.Run("with full answer", func(t *testing.T) {
			queryID := dns.Id()
			data := dnsGenNSReplySuccess(dnsGenQuery(dns.TypeNS, queryID), "ns1.zdns.google.")
			d := &DNSDecoderMiekg{}
			reply, err := d.DecodeNS(data, queryID)
			if err != nil {
				t.Fatal(err)
			}
			if len(reply) != 1 {
				t.Fatal("unexpected reply length")
			}
			if reply[0].Host != "ns1.zdns.google." {
				t.Fatal("unexpected reply host")
			}
		})
	})
}

// dnsGenQuery generates a query suitable to be used with testing.
func dnsGenQuery(qtype uint16, queryID uint16) []byte {
	question := dns.Question{
		Name:   dns.Fqdn("x.org"),
		Qtype:  qtype,
		Qclass: dns.ClassINET,
	}
	query := new(dns.Msg)
	query.Id = queryID
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.Question[0] = question
	data, err := query.Pack()
	runtimex.PanicOnError(err, "query.Pack failed")
	return data
}

// dnsGenReplyWithError generates a DNS reply for the given
// query type (e.g., dns.TypeA) using code as the Rcode.
func dnsGenReplyWithError(rawQuery []byte, code int) []byte {
	query := new(dns.Msg)
	err := query.Unpack(rawQuery)
	runtimex.PanicOnError(err, "query.Unpack failed")
	reply := new(dns.Msg)
	reply.Compress = true
	reply.MsgHdr.RecursionAvailable = true
	reply.SetRcode(query, code)
	data, err := reply.Pack()
	runtimex.PanicOnError(err, "reply.Pack failed")
	return data
}

// dnsGenLookupHostReplySuccess generates a successful DNS reply for the given
// qtype (e.g., dns.TypeA) containing the given ips... in the answer.
func dnsGenLookupHostReplySuccess(rawQuery []byte, ips ...string) []byte {
	query := new(dns.Msg)
	err := query.Unpack(rawQuery)
	runtimex.PanicOnError(err, "query.Unpack failed")
	runtimex.PanicIfFalse(len(query.Question) == 1, "more than one question")
	question := query.Question[0]
	runtimex.PanicIfFalse(
		question.Qtype == dns.TypeA || question.Qtype == dns.TypeAAAA,
		"invalid query type (expected A or AAAA)",
	)
	reply := new(dns.Msg)
	reply.Compress = true
	reply.MsgHdr.RecursionAvailable = true
	reply.SetReply(query)
	for _, ip := range ips {
		switch question.Qtype {
		case dns.TypeA:
			if isIPv6(ip) {
				continue
			}
			reply.Answer = append(reply.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn("x.org"),
					Rrtype: question.Qtype,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				A: net.ParseIP(ip),
			})
		case dns.TypeAAAA:
			if !isIPv6(ip) {
				continue
			}
			reply.Answer = append(reply.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn("x.org"),
					Rrtype: question.Qtype,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				AAAA: net.ParseIP(ip),
			})
		}
	}
	data, err := reply.Pack()
	runtimex.PanicOnError(err, "reply.Pack failed")
	return data
}

// dnsGenHTTPSReplySuccess generates a successful HTTPS response containing
// the given (possibly nil) alpns, ipv4s, and ipv6s.
func dnsGenHTTPSReplySuccess(rawQuery []byte, alpns, ipv4s, ipv6s []string) []byte {
	query := new(dns.Msg)
	err := query.Unpack(rawQuery)
	runtimex.PanicOnError(err, "query.Unpack failed")
	runtimex.PanicIfFalse(len(query.Question) == 1, "expected just a single question")
	question := query.Question[0]
	runtimex.PanicIfFalse(question.Qtype == dns.TypeHTTPS, "expected HTTPS query")
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
	runtimex.PanicOnError(err, "reply.Pack failed")
	return data
}

// dnsGenNSReplySuccess generates a successful NS reply using the given names.
func dnsGenNSReplySuccess(rawQuery []byte, names ...string) []byte {
	query := new(dns.Msg)
	err := query.Unpack(rawQuery)
	runtimex.PanicOnError(err, "query.Unpack failed")
	runtimex.PanicIfFalse(len(query.Question) == 1, "more than one question")
	question := query.Question[0]
	runtimex.PanicIfFalse(question.Qtype == dns.TypeNS, "expected NS query")
	reply := new(dns.Msg)
	reply.Compress = true
	reply.MsgHdr.RecursionAvailable = true
	reply.SetReply(query)
	for _, name := range names {
		reply.Answer = append(reply.Answer, &dns.NS{
			Hdr: dns.RR_Header{
				Name:   dns.Fqdn("x.org"),
				Rrtype: question.Qtype,
				Class:  dns.ClassINET,
				Ttl:    0,
			},
			Ns: name,
		})
	}
	data, err := reply.Pack()
	runtimex.PanicOnError(err, "reply.Pack failed")
	return data
}
