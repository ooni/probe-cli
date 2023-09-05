package filtering

import (
	"net"
	"strings"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func dnsComposeQuery(domain string, qtype uint16) *dns.Msg {
	question := dns.Question{
		Name:   dns.Fqdn(domain),
		Qtype:  qtype,
		Qclass: dns.ClassINET,
	}
	query := &dns.Msg{}
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.Question[0] = question
	query.Id = dns.Id()
	return query
}

// DNSComposeResponse composes a DNS response using the given IP addresses. If given no
// addresses, this function returns a successful, empty response. This function PANICS if
// the query argument does not contain EXACTLY one question.
//
// Deprecated: do not use this function in new code
func DNSComposeResponse(query *dns.Msg, ips ...net.IP) *dns.Msg {
	runtimex.PanicIfTrue(len(query.Question) != 1, "expecting a single question")
	question := query.Question[0]
	reply := new(dns.Msg)
	reply.Compress = true
	reply.MsgHdr.RecursionAvailable = true
	reply.SetReply(query)
	for _, ip := range ips {
		isIPv6 := strings.Contains(ip.String(), ":")
		if !isIPv6 && question.Qtype == dns.TypeA {
			reply.Answer = append(reply.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				A: ip,
			})
		} else if isIPv6 && question.Qtype == dns.TypeAAAA {
			reply.Answer = append(reply.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				AAAA: ip,
			})
		}
	}
	return reply
}
