package filtering

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// DNSAction is the action that this proxy should take.
type DNSAction int

const (
	// DNSActionProxy proxies the traffic to the upstream server.
	DNSActionProxy = DNSAction(iota)

	// DNSActionNXDOMAIN replies with NXDOMAIN.
	DNSActionNXDOMAIN

	// DNSActionRefused replies with Refused.
	DNSActionRefused

	// DNSActionLocalHost replies with `127.0.0.1` and `::1`.
	DNSActionLocalHost

	// DNSActionEmpty returns an empty reply.
	DNSActionEmpty

	// DNSActionTimeout never replies to the query.
	DNSActionTimeout
)

// DNSProxy is a DNS proxy that routes traffic to an upstream
// resolver and may implement filtering policies.
type DNSProxy struct {
	// OnQuery is the MANDATORY hook called whenever we
	// receive a query for the given domain.
	OnQuery func(domain string) DNSAction

	// Upstream is the OPTIONAL upstream transport.
	Upstream DNSTransport

	// mockableReply allows to mock DNSProxy.reply in tests.
	mockableReply func(query *dns.Msg) (*dns.Msg, error)
}

// DNSTransport is the type we expect from an upstream DNS transport.
type DNSTransport interface {
	RoundTrip(ctx context.Context, query []byte) ([]byte, error)
	CloseIdleConnections()
}

// DNSListener is the interface returned by DNSProxy.Start
type DNSListener interface {
	io.Closer
	LocalAddr() net.Addr
}

// Start starts the proxy.
func (p *DNSProxy) Start(address string) (DNSListener, error) {
	pconn, _, err := p.start(address)
	return pconn, err
}

func (p *DNSProxy) start(address string) (DNSListener, <-chan interface{}, error) {
	pconn, err := net.ListenPacket("udp", address)
	if err != nil {
		return nil, nil, err
	}
	done := make(chan interface{})
	go p.mainloop(pconn, done)
	return pconn, done, nil
}

func (p *DNSProxy) mainloop(pconn net.PacketConn, done chan<- interface{}) {
	defer close(done)
	for p.oneloop(pconn) {
		// nothing
	}
}

func (p *DNSProxy) oneloop(pconn net.PacketConn) bool {
	buffer := make([]byte, 1<<12)
	count, addr, err := pconn.ReadFrom(buffer)
	if err != nil {
		return !strings.HasSuffix(err.Error(), "use of closed network connection")
	}
	buffer = buffer[:count]
	query := &dns.Msg{}
	if err := query.Unpack(buffer); err != nil {
		return true // can continue
	}
	reply, err := p.reply(query)
	if err != nil {
		return true // can continue
	}
	replyBytes, err := reply.Pack()
	if err != nil {
		return true // can continue
	}
	pconn.WriteTo(replyBytes, addr)
	return true // can continue
}

func (p *DNSProxy) reply(query *dns.Msg) (*dns.Msg, error) {
	if p.mockableReply != nil {
		return p.mockableReply(query)
	}
	return p.replyDefault(query)
}

func (p *DNSProxy) replyDefault(query *dns.Msg) (*dns.Msg, error) {
	if len(query.Question) != 1 {
		return nil, errors.New("unhandled message")
	}
	name := query.Question[0].Name
	switch p.OnQuery(name) {
	case DNSActionProxy:
		return p.proxy(query)
	case DNSActionNXDOMAIN:
		return p.nxdomain(query), nil
	case DNSActionLocalHost:
		return p.localHost(query), nil
	case DNSActionEmpty:
		return p.empty(query), nil
	case DNSActionTimeout:
		return nil, errors.New("let's ignore this query")
	default:
		return p.refused(query), nil
	}
}

func (p *DNSProxy) refused(query *dns.Msg) *dns.Msg {
	m := new(dns.Msg)
	m.SetRcode(query, dns.RcodeRefused)
	return m
}

func (p *DNSProxy) nxdomain(query *dns.Msg) *dns.Msg {
	m := new(dns.Msg)
	m.SetRcode(query, dns.RcodeNameError)
	return m
}

func (p *DNSProxy) localHost(query *dns.Msg) *dns.Msg {
	return p.compose(query, net.IPv6loopback, net.IPv4(127, 0, 0, 1))
}

func (p *DNSProxy) empty(query *dns.Msg) *dns.Msg {
	return p.compose(query)
}

func (p *DNSProxy) compose(query *dns.Msg, ips ...net.IP) *dns.Msg {
	runtimex.PanicIfTrue(len(query.Question) != 1, "expecting a single question")
	question := query.Question[0]
	reply := new(dns.Msg)
	reply.Compress = true
	reply.MsgHdr.RecursionAvailable = true
	reply.SetReply(query)
	for _, ip := range ips {
		isIPv6 := strings.Contains(ip.String(), ":")
		if !isIPv6 && question.Qtype == dns.TypeA {
			reply.Answer = append(reply.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				A: ip,
			})
		} else if isIPv6 && question.Qtype == dns.TypeAAAA {
			reply.Answer = append(reply.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				AAAA: ip,
			})
		}
	}
	return reply
}

func (p *DNSProxy) proxy(query *dns.Msg) (*dns.Msg, error) {
	queryBytes, err := query.Pack()
	if err != nil {
		return nil, err
	}
	txp := p.dnstransport()
	defer txp.CloseIdleConnections()
	ctx := context.Background()
	replyBytes, err := txp.RoundTrip(ctx, queryBytes)
	if err != nil {
		return nil, err
	}
	reply := &dns.Msg{}
	if err := reply.Unpack(replyBytes); err != nil {
		return nil, err
	}
	return reply, nil
}

func (p *DNSProxy) dnstransport() DNSTransport {
	if p.Upstream != nil {
		return p.Upstream
	}
	const URL = "https://1.1.1.1/dns-query"
	return netxlite.NewDNSOverHTTPS(http.DefaultClient, URL)
}
