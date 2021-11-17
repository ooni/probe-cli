package netxlite

import "github.com/miekg/dns"

// The DNSDecoder decodes DNS replies.
type DNSDecoder interface {
	// DecodeLookupHost decodes an A or AAAA reply.
	//
	// Arguments:
	//
	// - qtype is the query type (e.g., dns.TypeAAAA)
	//
	// - data contains the reply bytes read from a DNSTransport
	//
	// Returns:
	//
	// - on success, a list of IP addrs inside the reply and a nil error
	//
	// - on failure, a nil list and an error.
	//
	// Note that this function will return an error if there is no
	// IP address inside of the reply.
	DecodeLookupHost(qtype uint16, data []byte) ([]string, error)

	// DecodeHTTPS decodes an HTTPS reply.
	//
	// The argument is the reply as read by the DNSTransport.
	//
	// On success, this function returns an HTTPSSvc structure and
	// a nil error. On failure, the HTTPSSvc pointer is nil and
	// the error points to the error that occurred.
	//
	// This function will return an error if the HTTPS reply does not
	// contain at least a valid ALPN entry. It will not return
	// an error, though, when there are no IPv4/IPv6 hints in the reply.
	DecodeHTTPS(data []byte) (*HTTPSSvc, error)
}

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
	default:
		return nil, ErrOODNSMisbehaving
	}
}

func (d *DNSDecoderMiekg) DecodeHTTPS(data []byte) (*HTTPSSvc, error) {
	reply, err := d.parseReply(data)
	if err != nil {
		return nil, err
	}
	out := &HTTPSSvc{}
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
	if len(out.ALPN) <= 0 {
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

var _ DNSDecoder = &DNSDecoderMiekg{}
