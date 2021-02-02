package resolver

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func GenReplyError(t *testing.T, code int) []byte {
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

func GenReplySuccess(t *testing.T, qtype uint16, ips ...string) []byte {
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
