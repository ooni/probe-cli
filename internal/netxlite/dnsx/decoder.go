package dnsx

import (
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// HTTPSSvc is an HTTPSSvc reply.
type HTTPSSvc = model.HTTPSSvc

type https struct {
	alpn     []string
	ipv4hint []string
	ipv6hint []string
}

var _ HTTPSSvc = &https{}

func (h *https) ALPN() []string {
	return h.alpn
}

func (h *https) IPv4Hint() []string {
	return h.ipv4hint
}

func (h *https) IPv6Hint() []string {
	return h.ipv6hint
}

// The Decoder decodes a DNS replies.
type Decoder interface {
	// DecodeLookupHost decodes an A or AAAA reply.
	DecodeLookupHost(qtype uint16, data []byte) ([]string, error)

	// DecodeHTTPS decodes an HTTPS reply.
	DecodeHTTPS(data []byte) (HTTPSSvc, error)
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

func (d *MiekgDecoder) DecodeHTTPS(data []byte) (HTTPSSvc, error) {
	reply, err := d.parseReply(data)
	if err != nil {
		return nil, err
	}
	out := &https{}
	for _, answer := range reply.Answer {
		switch avalue := answer.(type) {
		case *dns.HTTPS:
			for _, v := range avalue.Value {
				switch extv := v.(type) {
				case *dns.SVCBAlpn:
					out.alpn = extv.Alpn
				case *dns.SVCBIPv4Hint:
					for _, ip := range extv.Hint {
						out.ipv4hint = append(out.ipv4hint, ip.String())
					}
				case *dns.SVCBIPv6Hint:
					for _, ip := range extv.Hint {
						out.ipv6hint = append(out.ipv6hint, ip.String())
					}
				}
			}
		}
	}
	if len(out.alpn) <= 0 {
		return nil, errorsx.ErrOODNSNoAnswer
	}
	return out, nil
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
