package netxlite

import (
	"bytes"
	"errors"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestDNSDecoderMiekg(t *testing.T) {
	t.Run("DecodeResponse", func(t *testing.T) {
		t.Run("UnpackError", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			resp, err := d.DecodeResponse(nil, &mocks.DNSQuery{})
			if err == nil || err.Error() != "dns: overflow unpacking uint16" {
				t.Fatal("unexpected error", err)
			}
			if resp != nil {
				t.Fatal("expected nil resp here")
			}
		})

		t.Run("with bytes containing a query", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			rawQuery := dnsGenQuery(dns.TypeA, queryID)
			resp, err := d.DecodeResponse(rawQuery, &mocks.DNSQuery{})
			if !errors.Is(err, ErrDNSIsQuery) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("expected nil resp here")
			}
		})

		t.Run("wrong query ID", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			const (
				queryID     = 17
				unrelatedID = 14
			)
			reply := dnsGenLookupHostReplySuccess(dnsGenQuery(dns.TypeA, queryID))
			resp, err := d.DecodeResponse(reply, &mocks.DNSQuery{
				MockID: func() uint16 {
					return unrelatedID
				},
			})
			if !errors.Is(err, ErrDNSReplyWithWrongQueryID) {
				t.Fatal("unexpected error", err)
			}
			if resp != nil {
				t.Fatal("expected nil resp here")
			}
		})

		t.Run("dnsResponse.Query", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			rawQuery := dnsGenQuery(dns.TypeA, queryID)
			rawResponse := dnsGenLookupHostReplySuccess(rawQuery)
			query := &mocks.DNSQuery{
				MockID: func() uint16 {
					return queryID
				},
			}
			resp, err := d.DecodeResponse(rawResponse, query)
			if err != nil {
				t.Fatal(err)
			}
			if resp.Query().ID() != query.ID() {
				t.Fatal("invalid query")
			}
		})

		t.Run("dnsResponse.Bytes", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			rawQuery := dnsGenQuery(dns.TypeA, queryID)
			rawResponse := dnsGenLookupHostReplySuccess(rawQuery)
			query := &mocks.DNSQuery{
				MockID: func() uint16 {
					return queryID
				},
			}
			resp, err := d.DecodeResponse(rawResponse, query)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(rawResponse, resp.Bytes()) {
				t.Fatal("invalid bytes")
			}
		})

		t.Run("dnsResponse.Rcode", func(t *testing.T) {
			d := &DNSDecoderMiekg{}
			queryID := dns.Id()
			rawQuery := dnsGenQuery(dns.TypeA, queryID)
			rawResponse := dnsGenReplyWithError(rawQuery, dns.RcodeRefused)
			query := &mocks.DNSQuery{
				MockID: func() uint16 {
					return queryID
				},
			}
			resp, err := d.DecodeResponse(rawResponse, query)
			if err != nil {
				t.Fatal(err)
			}
			if resp.Rcode() != dns.RcodeRefused {
				t.Fatal("invalid rcode")
			}
		})

		t.Run("dnsResponse.rcodeToError", func(t *testing.T) {
			// Here we want to ensure we map all the errors we recognize
			// correctly and we also map unrecognized errors correctly
			var inputsOutputs = []struct {
				name  string
				rcode int
				err   error
			}{{
				name:  "when rcode is zero",
				rcode: 0,
				err:   nil,
			}, {
				name:  "NXDOMAIN",
				rcode: dns.RcodeNameError,
				err:   ErrOODNSNoSuchHost,
			}, {
				name:  "refused",
				rcode: dns.RcodeRefused,
				err:   ErrOODNSRefused,
			}, {
				name:  "servfail",
				rcode: dns.RcodeServerFailure,
				err:   ErrOODNSServfail,
			}, {
				name:  "anything else",
				rcode: dns.RcodeFormatError,
				err:   ErrOODNSMisbehaving,
			}}
			for _, io := range inputsOutputs {
				t.Run(io.name, func(t *testing.T) {
					d := &DNSDecoderMiekg{}
					queryID := dns.Id()
					rawQuery := dnsGenQuery(dns.TypeHTTPS, queryID)
					rawResponse := dnsGenReplyWithError(rawQuery, io.rcode)
					query := &mocks.DNSQuery{
						MockID: func() uint16 {
							return queryID
						},
					}
					resp, err := d.DecodeResponse(rawResponse, query)
					if err != nil {
						t.Fatal(err)
					}
					// The following cast should always work in this configuration
					err = resp.(*dnsResponse).rcodeToError()
					if !errors.Is(err, io.err) {
						t.Fatal("unexpected err", err)
					}
				})
			}
		})

		t.Run("dnsResponse.DecodeHTTPS", func(t *testing.T) {
			t.Run("with failure", func(t *testing.T) {
				// Ensure that we're not trying to decode if rcode != 0
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeHTTPS, queryID)
				rawResponse := dnsGenReplyWithError(rawQuery, dns.RcodeRefused)
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				https, err := resp.DecodeHTTPS()
				if !errors.Is(err, ErrOODNSRefused) {
					t.Fatal("unexpected err", err)
				}
				if https != nil {
					t.Fatal("expected nil https result")
				}
			})

			t.Run("with empty answer", func(t *testing.T) {
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeHTTPS, queryID)
				rawResponse := dnsGenHTTPSReplySuccess(rawQuery, nil, nil, nil)
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				https, err := resp.DecodeHTTPS()
				if !errors.Is(err, ErrOODNSNoAnswer) {
					t.Fatal("unexpected err", err)
				}
				if https != nil {
					t.Fatal("expected nil https results")
				}
			})

			t.Run("with full answer", func(t *testing.T) {
				alpn := []string{"h3"}
				v4 := []string{"1.1.1.1"}
				v6 := []string{"::1"}
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeHTTPS, queryID)
				rawResponse := dnsGenHTTPSReplySuccess(rawQuery, alpn, v4, v6)
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				reply, err := resp.DecodeHTTPS()
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

		t.Run("dnsResponse.DecodeNS", func(t *testing.T) {
			t.Run("with failure", func(t *testing.T) {
				// Ensure that we're not trying to decode if rcode != 0
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeNS, queryID)
				rawResponse := dnsGenReplyWithError(rawQuery, dns.RcodeRefused)
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				ns, err := resp.DecodeNS()
				if !errors.Is(err, ErrOODNSRefused) {
					t.Fatal("unexpected err", err)
				}
				if len(ns) > 0 {
					t.Fatal("expected empty ns result")
				}
			})

			t.Run("with empty answer", func(t *testing.T) {
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeNS, queryID)
				rawResponse := dnsGenNSReplySuccess(rawQuery)
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				ns, err := resp.DecodeNS()
				if !errors.Is(err, ErrOODNSNoAnswer) {
					t.Fatal("unexpected err", err)
				}
				if len(ns) > 0 {
					t.Fatal("expected empty ns results")
				}
			})

			t.Run("with full answer", func(t *testing.T) {
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeNS, queryID)
				rawResponse := dnsGenNSReplySuccess(rawQuery, "ns1.zdns.google.")
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				ns, err := resp.DecodeNS()
				if err != nil {
					t.Fatal(err)
				}
				if len(ns) != 1 {
					t.Fatal("unexpected ns length")
				}
				if ns[0].Host != "ns1.zdns.google." {
					t.Fatal("unexpected host")
				}
			})
		})

		t.Run("dnsResponse.LookupHost", func(t *testing.T) {
			t.Run("with failure", func(t *testing.T) {
				// Ensure that we're not trying to decode if rcode != 0
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeA, queryID)
				rawResponse := dnsGenReplyWithError(rawQuery, dns.RcodeRefused)
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				addrs, err := resp.DecodeLookupHost()
				if !errors.Is(err, ErrOODNSRefused) {
					t.Fatal("unexpected err", err)
				}
				if len(addrs) > 0 {
					t.Fatal("expected empty addrs result")
				}
			})

			t.Run("with empty answer", func(t *testing.T) {
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeA, queryID)
				rawResponse := dnsGenLookupHostReplySuccess(rawQuery)
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				addrs, err := resp.DecodeLookupHost()
				if !errors.Is(err, ErrOODNSNoAnswer) {
					t.Fatal("unexpected err", err)
				}
				if len(addrs) > 0 {
					t.Fatal("expected empty ns results")
				}
			})

			t.Run("decode A", func(t *testing.T) {
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeA, queryID)
				rawResponse := dnsGenLookupHostReplySuccess(rawQuery, "1.1.1.1", "8.8.8.8")
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
					MockType: func() uint16 {
						return dns.TypeA
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				addrs, err := resp.DecodeLookupHost()
				if err != nil {
					t.Fatal(err)
				}
				if len(addrs) != 2 {
					t.Fatal("expected two entries here")
				}
				if addrs[0] != "1.1.1.1" {
					t.Fatal("invalid first IPv4 entry")
				}
				if addrs[1] != "8.8.8.8" {
					t.Fatal("invalid second IPv4 entry")
				}
			})

			t.Run("decode AAAA", func(t *testing.T) {
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeAAAA, queryID)
				rawResponse := dnsGenLookupHostReplySuccess(rawQuery, "::1", "fe80::1")
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
					MockType: func() uint16 {
						return dns.TypeAAAA
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				addrs, err := resp.DecodeLookupHost()
				if err != nil {
					t.Fatal(err)
				}
				if len(addrs) != 2 {
					t.Fatal("expected two entries here")
				}
				if addrs[0] != "::1" {
					t.Fatal("invalid first IPv6 entry")
				}
				if addrs[1] != "fe80::1" {
					t.Fatal("invalid second IPv6 entry")
				}
			})

			t.Run("unexpected A reply to AAAA query", func(t *testing.T) {
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeAAAA, queryID)
				rawResponse := dnsGenLookupHostReplySuccess(rawQuery, "1.1.1.1", "8.8.8.8")
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
					MockType: func() uint16 {
						return dns.TypeAAAA
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				addrs, err := resp.DecodeLookupHost()
				if !errors.Is(err, ErrOODNSNoAnswer) {
					t.Fatal("not the error we expected", err)
				}
				if len(addrs) > 0 {
					t.Fatal("expected no addrs here")
				}
			})

			t.Run("unexpected AAAA reply to A query", func(t *testing.T) {
				d := &DNSDecoderMiekg{}
				queryID := dns.Id()
				rawQuery := dnsGenQuery(dns.TypeA, queryID)
				rawResponse := dnsGenLookupHostReplySuccess(rawQuery, "::1", "fe80::1")
				query := &mocks.DNSQuery{
					MockID: func() uint16 {
						return queryID
					},
					MockType: func() uint16 {
						return dns.TypeA
					},
				}
				resp, err := d.DecodeResponse(rawResponse, query)
				if err != nil {
					t.Fatal(err)
				}
				addrs, err := resp.DecodeLookupHost()
				if !errors.Is(err, ErrOODNSNoAnswer) {
					t.Fatal("not the error we expected", err)
				}
				if len(addrs) > 0 {
					t.Fatal("expected no addrs here")
				}
			})
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

// dnsGenLookupHostReplySuccess generates a successful DNS reply containing the given ips...
// in the answers where each answer's type depends on the IP's type (A/AAAA).
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
		switch isIPv6(ip) {
		case false:
			reply.Answer = append(reply.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				A: net.ParseIP(ip),
			})
		case true:
			reply.Answer = append(reply.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeAAAA,
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
