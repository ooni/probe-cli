package netplumbing

// This file contains the implementation of Transport's DNS decoding functions.

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
	if err := txp.dnsDecodeRcode(reply); err != nil {
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
		return nil, errors.New("netplumbing: no IP address returned")
	}
	return addrs, nil
}

func (txp *Transport) dnsDecodeRcode(reply *dns.Msg) error {
	// TODO(bassosimone): map more errors to net.DNSError names
	switch reply.Rcode {
	case dns.RcodeSuccess:
		return nil
	case dns.RcodeNameError:
		return errors.New("netplumbing: no such host")
	default:
		return errors.New("netplumbing: server misbehaving")
	}
}

// DNSDecodeCNAME decodes an CNAME reply returning the CNAME.
func (txp *Transport) DNSDecodeCNAME(reply *dns.Msg) (string, error) {
	if err := txp.dnsDecodeRcode(reply); err != nil {
		return "", err
	}
	var cname string
	for _, answer := range reply.Answer {
		if rrcname, ok := answer.(*dns.CNAME); ok {
			cname = rrcname.Target
		}
	}
	if cname == "" {
		return "", errors.New("netplumbing: no CNAME returned")
	}
	return cname, nil
}
