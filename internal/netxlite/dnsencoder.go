package netxlite

import "github.com/miekg/dns"

// The DNSEncoder encodes DNS queries to bytes
type DNSEncoder interface {
	// Encode transforms its arguments into a serialized DNS query.
	//
	// Arguments:
	//
	// - domain is the domain for the query (e.g., x.org);
	//
	// - qtype is the query type (e.g., dns.TypeA);
	//
	// - padding is whether to add padding to the query.
	//
	// On success, this function returns a valid byte array and
	// a nil error. On failure, we have an error and the byte array is nil.
	Encode(domain string, qtype uint16, padding bool) ([]byte, error)
}

// DNSEncoderMiekg uses github.com/miekg/dns to implement the Encoder.
type DNSEncoderMiekg struct{}

const (
	// dnsPaddingDesiredBlockSize is the size that the padded query should be multiple of
	dnsPaddingDesiredBlockSize = 128

	// dnsEDNS0MaxResponseSize is the maximum response size for EDNS0
	dnsEDNS0MaxResponseSize = 4096

	// dnsDNSSECEnabled turns on support for DNSSEC when using EDNS0
	dnsDNSSECEnabled = true
)

func (e *DNSEncoderMiekg) Encode(domain string, qtype uint16, padding bool) ([]byte, error) {
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
		query.SetEdns0(dnsEDNS0MaxResponseSize, dnsDNSSECEnabled)
		// Clients SHOULD pad queries to the closest multiple of
		// 128 octets RFC8467#section-4.1. We inflate the query
		// length by the size of the option (i.e. 4 octets). The
		// cast to uint is necessary to make the modulus operation
		// work as intended when the desiredBlockSize is smaller
		// than (query.Len()+4) ¯\_(ツ)_/¯.
		remainder := (dnsPaddingDesiredBlockSize - uint(query.Len()+4)) % dnsPaddingDesiredBlockSize
		opt := new(dns.EDNS0_PADDING)
		opt.Padding = make([]byte, remainder)
		query.IsEdns0().Option = append(query.IsEdns0().Option, opt)
	}
	return query.Pack()
}

var _ DNSEncoder = &DNSEncoderMiekg{}
