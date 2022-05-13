package netxlite

import (
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSDecoderMiekg uses github.com/miekg/dns to implement the Decoder.
type DNSDecoderMiekg struct{}

func (d *DNSDecoderMiekg) parseReply(data []byte) (*dns.Msg, error) {
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
		return nil, ErrOODNSNoSuchHost
	case dns.RcodeRefused:
		return nil, ErrOODNSRefused
	case dns.RcodeServerFailure:
		return nil, ErrOODNSServfail
	default:
		return nil, ErrOODNSMisbehaving
	}
}

func (d *DNSDecoderMiekg) DecodeHTTPS(data []byte) (*model.HTTPSSvc, error) {
	reply, err := d.parseReply(data)
	if err != nil {
		return nil, err
	}
	out := &model.HTTPSSvc{
		ALPN: []string{}, // ensure it's not nil
		IPv4: []string{}, // ensure it's not nil
		IPv6: []string{}, // ensure it's not nil
	}
	for _, answer := range reply.Answer {
		switch avalue := answer.(type) {
		case *dns.HTTPS:
			for _, v := range avalue.Value {
				switch extv := v.(type) {
				case *dns.SVCBAlpn:
					out.ALPN = extv.Alpn
				case *dns.SVCBIPv4Hint:
					for _, ip := range extv.Hint {
						out.IPv4 = append(out.IPv4, ip.String())
					}
				case *dns.SVCBIPv6Hint:
					for _, ip := range extv.Hint {
						out.IPv6 = append(out.IPv6, ip.String())
					}
				}
			}
		}
	}
	if len(out.IPv4) <= 0 && len(out.IPv6) <= 0 {
		return nil, ErrOODNSNoAnswer
	}
	return out, nil
}

func (d *DNSDecoderMiekg) DecodeLookupHost(qtype uint16, data []byte) ([]string, error) {
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
		return nil, ErrOODNSNoAnswer
	}
	return addrs, nil
}

var _ model.DNSDecoder = &DNSDecoderMiekg{}
