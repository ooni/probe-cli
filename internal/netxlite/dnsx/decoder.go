package dnsx

import (
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// The Decoder decodes DNS replies.
type Decoder interface {
	// DecodeLookupHost decodes an A or AAAA reply.
	DecodeLookupHost(qtype uint16, data []byte) ([]string, error)
}

// MiekgDecoder uses github.com/miekg/dns to implement the Decoder.
type MiekgDecoder struct{}

func (d *MiekgDecoder) parseReply(data []byte) (*dns.Msg, error) {
	reply := new(dns.Msg)
	if err := reply.Unpack(data); err != nil {
		return nil, err
	}
	// TODO(bassosimone): map more errors to net.DNSError names
	// TODO(bassosimone): add support for lame referral.
	switch reply.Rcode {
	case dns.RcodeSuccess:
		return reply, nil
	case dns.RcodeNameError:
		return nil, errorsx.ErrOODNSNoSuchHost
	case dns.RcodeRefused:
		return nil, errorsx.ErrOODNSRefused
	default:
		return nil, errorsx.ErrOODNSMisbehaving
	}
}

func (d *MiekgDecoder) DecodeLookupHost(qtype uint16, data []byte) ([]string, error) {
	reply, err := d.parseReply(data)
	if err != nil {
		return nil, err
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
		return nil, errorsx.ErrOODNSNoAnswer
	}
	return addrs, nil
}

var _ Decoder = &MiekgDecoder{}
