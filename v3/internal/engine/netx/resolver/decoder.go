package resolver

import (
	"errors"

	"github.com/miekg/dns"
)

// The Decoder decodes a DNS reply into A or AAAA entries. It will use the
// provided qtype and only look for mathing entries. It will return error if
// there are no entries for the requested qtype inside the reply.
type Decoder interface {
	Decode(qtype uint16, data []byte) ([]string, error)
}

// MiekgDecoder uses github.com/miekg/dns to implement the Decoder.
type MiekgDecoder struct{}

// Decode implements Decoder.Decode.
func (d MiekgDecoder) Decode(qtype uint16, data []byte) ([]string, error) {
	reply := new(dns.Msg)
	if err := reply.Unpack(data); err != nil {
		return nil, err
	}
	// TODO(bassosimone): map more errors to net.DNSError names
	switch reply.Rcode {
	case dns.RcodeSuccess:
	case dns.RcodeNameError:
		return nil, errors.New("ooniresolver: no such host")
	default:
		return nil, errors.New("ooniresolver: query failed")
	}
	var addrs []string
	for _, answer := range reply.Answer {
		switch qtype {
		case dns.TypeA:
			if rra, ok := answer.(*dns.A); ok {
				ip := rra.A
				addrs = append(addrs, ip.String())
			}
		case dns.TypeAAAA:
			if rra, ok := answer.(*dns.AAAA); ok {
				ip := rra.AAAA
				addrs = append(addrs, ip.String())
			}
		}
	}
	if len(addrs) <= 0 {
		return nil, errors.New("ooniresolver: no response returned")
	}
	return addrs, nil
}

var _ Decoder = MiekgDecoder{}
