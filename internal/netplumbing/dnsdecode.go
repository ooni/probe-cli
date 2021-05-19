package netplumbing

import (
	"errors"

	"github.com/miekg/dns"
)

// DNSDecodeA decodes an A reply returning the IP addresses.
func (txp *Transport) DNSDecodeA(reply *dns.Msg) ([]string, error) {
	return txp.dnsDecodeSomeA(reply, dns.TypeA)
}

// DNSDecodeAAAA decodes an AAAA reply returning the IP addresses.
func (txp *Transport) DNSDecodeAAAA(reply *dns.Msg) ([]string, error) {
	return txp.dnsDecodeSomeA(reply, dns.TypeAAAA)
}

// dnsDecodeSomeA decodes an A or AAAA reply returning the IP addresses.
func (txp *Transport) dnsDecodeSomeA(reply *dns.Msg, qtype uint16) ([]string, error) {
	// TODO(bassosimone): map more errors to net.DNSError names
	switch reply.Rcode {
	case dns.RcodeSuccess:
	case dns.RcodeNameError:
		return nil, errors.New("netplumbing: no such host")
	default:
		return nil, errors.New("netplumbing: server misbehaving")
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
		return nil, errors.New("netplumbing: no response returned")
	}
	return addrs, nil
}
