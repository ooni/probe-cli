package netxlite

//
// Encode DNS queries to byte arrays
//

import (
	"sync"
	"sync/atomic"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
)

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

// Encoder implements model.DNSEncoder.Encode.
func (e *DNSEncoderMiekg) Encode(domain string, qtype uint16, padding bool) model.DNSQuery {
	return &dnsQuery{
		bytesCalls:    &atomic.Int64{},
		domain:        domain,
		kind:          qtype,
		id:            dns.Id(),
		memoizedBytes: []byte{},
		mu:            sync.Mutex{},
		padding:       padding,
	}
}

// dnsQuery implements model.DNSQuery.
type dnsQuery struct {
	// bytesCalls counts the calls to the bytes() method
	bytesCalls *atomic.Int64

	// domain is the domain.
	domain string

	// kind is the query type.
	kind uint16

	// id is the query ID.
	id uint16

	// memoizedBytes contains the query encoded as bytes. We only fill
	// this field the first time the Bytes method is called.
	memoizedBytes []byte

	// mu provides mutual exclusion.
	mu sync.Mutex

	// padding indicates whether we need padding.
	padding bool
}

// Domain implements model.DNSQuery.Domain.
func (q *dnsQuery) Domain() string {
	return q.domain
}

// Type implements model.DNSQuery.Type.
func (q *dnsQuery) Type() uint16 {
	return q.kind
}

// Bytes implements model.DNSQuery.Bytes.
func (q *dnsQuery) Bytes() ([]byte, error) {
	defer q.mu.Unlock()
	q.mu.Lock()
	if len(q.memoizedBytes) <= 0 {
		q.bytesCalls.Add(1) // for testing
		data, err := q.bytes()
		if err != nil {
			return nil, err
		}
		q.memoizedBytes = data
	}
	return q.memoizedBytes, nil
}

// bytes is the unmemoized implementation of Bytes
func (q *dnsQuery) bytes() ([]byte, error) {
	question := dns.Question{
		Name:   dns.Fqdn(q.domain),
		Qtype:  q.kind,
		Qclass: dns.ClassINET,
	}
	query := new(dns.Msg)
	query.Id = q.id
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.Question[0] = question
	if q.padding {
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

// ID implements model.DNSQuery.ID
func (q *dnsQuery) ID() uint16 {
	return q.id
}

var _ model.DNSEncoder = &DNSEncoderMiekg{}
var _ model.DNSQuery = &dnsQuery{}
