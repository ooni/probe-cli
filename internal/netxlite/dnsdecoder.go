package netxlite

//
// Decode byte arrays to DNS messages
//

import (
	"errors"
	"net"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSDecoderMiekg uses github.com/miekg/dns to implement the Decoder.
type DNSDecoderMiekg struct{}

var (
	// ErrDNSReplyWithWrongQueryID indicates we have got a DNS reply with the wrong queryID.
	ErrDNSReplyWithWrongQueryID = errors.New(FailureDNSReplyWithWrongQueryID)

	// ErrDNSIsQuery indicates that we were passed a DNS query.
	ErrDNSIsQuery = errors.New("ooresolver: expected response but received query")
)

// DecodeResponse implements model.DNSDecoder.DecodeResponse.
func (d *DNSDecoderMiekg) DecodeResponse(data []byte, query model.DNSQuery) (model.DNSResponse, error) {
	reply := &dns.Msg{}
	if err := reply.Unpack(data); err != nil {
		return nil, err
	}
	if !reply.Response {
		return nil, ErrDNSIsQuery
	}
	if reply.Id != query.ID() {
		return nil, ErrDNSReplyWithWrongQueryID
	}
	resp := &dnsResponse{
		bytes: data,
		msg:   reply,
		query: query,
	}
	return resp, nil
}

// dnsResponse implements model.DNSResponse.
type dnsResponse struct {
	// bytes contains the response bytes.
	bytes []byte

	// msg contains the message.
	msg *dns.Msg

	// query is the original query.
	query model.DNSQuery
}

// Query implements model.DNSResponse.Query.
func (r *dnsResponse) Query() model.DNSQuery {
	return r.query
}

// Bytes implements model.DNSResponse.Bytes.
func (r *dnsResponse) Bytes() []byte {
	return r.bytes
}

// Rcode implements model.DNSResponse.Rcode.
func (r *dnsResponse) Rcode() int {
	return r.msg.Rcode
}

func (r *dnsResponse) rcodeToError() error {
	// TODO(bassosimone): map more errors to net.DNSError names
	// TODO(bassosimone): add support for lame referral.
	switch r.msg.Rcode {
	case dns.RcodeSuccess:
		return nil
	case dns.RcodeNameError:
		return ErrOODNSNoSuchHost
	case dns.RcodeRefused:
		return ErrOODNSRefused
	case dns.RcodeServerFailure:
		return ErrOODNSServfail
	default:
		return ErrOODNSMisbehaving
	}
}

// DecodeHTTPS implements model.DNSResponse.DecodeHTTPS.
func (r *dnsResponse) DecodeHTTPS() (*model.HTTPSSvc, error) {
	if err := r.rcodeToError(); err != nil {
		return nil, err
	}
	out := &model.HTTPSSvc{
		ALPN: []string{}, // ensure it's not nil
		IPv4: []string{}, // ensure it's not nil
		IPv6: []string{}, // ensure it's not nil
	}
	for _, answer := range r.msg.Answer {
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

// DecodeLookupHost implements model.DNSResponse.DecodeLookupHost.
func (r *dnsResponse) DecodeLookupHost() ([]string, error) {
	if err := r.rcodeToError(); err != nil {
		return nil, err
	}
	var addrs []string
	for _, answer := range r.msg.Answer {
		switch r.Query().Type() {
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

// DecodeNS implements model.DNSResponse.DecodeNS.
func (r *dnsResponse) DecodeNS() ([]*net.NS, error) {
	if err := r.rcodeToError(); err != nil {
		return nil, err
	}
	out := []*net.NS{}
	for _, answer := range r.msg.Answer {
		switch avalue := answer.(type) {
		case *dns.NS:
			out = append(out, &net.NS{Host: avalue.Ns})
		}
	}
	if len(out) < 1 {
		return nil, ErrOODNSNoAnswer
	}
	return out, nil
}

var _ model.DNSDecoder = &DNSDecoderMiekg{}
var _ model.DNSResponse = &dnsResponse{}
