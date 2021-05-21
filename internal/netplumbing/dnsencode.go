package netplumbing

// This file contains the implementation of Transport's DNS encoding functions.

import "github.com/miekg/dns"

// DNSEncodeA encodes an A query. The padding argument indicates
// whether you want to generate a query using padding.
func (txp *Transport) DNSEncodeA(domain string, padding bool) *dns.Msg {
	return txp.dnsEncode(domain, padding, dns.TypeA)
}

// DNSEncodeA encodes an AAAA query. The padding argument indicates
// whether you want to generate a query using padding.
func (txp *Transport) DNSEncodeAAAA(domain string, padding bool) *dns.Msg {
	return txp.dnsEncode(domain, padding, dns.TypeAAAA)
}

// DNSEncodeCNAME encodes an CNAME query. The padding argument indicates
// whether you want to generate a query using padding.
func (txp *Transport) DNSEncodeCNAME(domain string, padding bool) *dns.Msg {
	return txp.dnsEncode(domain, padding, dns.TypeCNAME)
}

// dnsEncode encodes a DNS query.
func (txp *Transport) dnsEncode(
	domain string, padding bool, qtype uint16) *dns.Msg {
	const (
		// paddingDesiredBlockSize is the size that the padded query
		// should be multiple of
		paddingDesiredBlockSize = 128
		// EDNS0MaxResponseSize is the maximum response size for EDNS0
		EDNS0MaxResponseSize = 4096
		// DNSSECEnabled turns on support for DNSSEC when using EDNS0
		DNSSECEnabled = true
	)
	question := dns.Question{
		Name:   dns.Fqdn(domain),
		Qtype:  qtype,
		Qclass: dns.ClassINET,
	}
	query := new(dns.Msg)
	query.Id = dns.Id()
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.Question[0] = question
	if padding {
		query.SetEdns0(EDNS0MaxResponseSize, DNSSECEnabled)
		// Clients SHOULD pad queries to the closest multiple of
		// 128 octets RFC8467#section-4.1. We inflate the query
		// length by the size of the option (i.e. 4 octets). The
		// cast to uint is necessary to make the modulus operation
		// work as intended when the desiredBlockSize is smaller
		// than (query.Len()+4) ¯\_(ツ)_/¯.
		remainder := (paddingDesiredBlockSize - uint(query.Len()+4)) % paddingDesiredBlockSize
		opt := new(dns.EDNS0_PADDING)
		opt.Padding = make([]byte, remainder)
		query.IsEdns0().Option = append(query.IsEdns0().Option, opt)
	}
	return query
}
