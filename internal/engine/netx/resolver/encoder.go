package resolver

import "github.com/miekg/dns"

// The Encoder encodes DNS queries to bytes
type Encoder interface {
	Encode(domain string, qtype uint16, padding bool) ([]byte, error)
}

// MiekgEncoder uses github.com/miekg/dns to implement the Encoder.
type MiekgEncoder struct{}

const (
	// PaddingDesiredBlockSize is the size that the padded query should be multiple of
	PaddingDesiredBlockSize = 128

	// EDNS0MaxResponseSize is the maximum response size for EDNS0
	EDNS0MaxResponseSize = 4096

	// DNSSECEnabled turns on support for DNSSEC when using EDNS0
	DNSSECEnabled = true
)

// Encode implements Encoder.Encode
func (e MiekgEncoder) Encode(domain string, qtype uint16, padding bool) ([]byte, error) {
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
		remainder := (PaddingDesiredBlockSize - uint(query.Len()+4)) % PaddingDesiredBlockSize
		opt := new(dns.EDNS0_PADDING)
		opt.Padding = make([]byte, remainder)
		query.IsEdns0().Option = append(query.IsEdns0().Option, opt)
	}
	return query.Pack()
}

var _ Encoder = MiekgEncoder{}
