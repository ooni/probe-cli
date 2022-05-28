package filtering

import (
	"io"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// DNSAction is a DNS filtering action that a DNSServer should take.
type DNSAction string

const (
	// DNSActionNXDOMAIN replies with NXDOMAIN.
	DNSActionNXDOMAIN = DNSAction("nxdomain")

	// DNSActionRefused replies with Refused.
	DNSActionRefused = DNSAction("refused")

	// DNSActionLocalHost replies with `127.0.0.1` and `::1`.
	DNSActionLocalHost = DNSAction("localhost")

	// DNSActionNoAnswer returns an empty reply.
	DNSActionNoAnswer = DNSAction("no-answer")

	// DNSActionTimeout never replies to the query.
	DNSActionTimeout = DNSAction("timeout")

	// DNSActionCache causes the server to check the cache. If there
	// are entries, they are returned. Otherwise, NXDOMAIN is returned.
	DNSActionCache = DNSAction("cache")

	// DNSActionLocalHostPlusCache combines the LocalHost and
	// Cache actions returning first a localhost response followed
	// by a subsequent response obtained using the cache.
	DNSActionLocalHostPlusCache = DNSAction("localhost+cache")
)

// DNSServer is a DNS server implementing filtering policies.
type DNSServer struct {
	// Cache is the OPTIONAL DNS cache. Note that the keys of the map
	// must be FQDNs (i.e., including the final `.`).
	Cache map[string][]string

	// OnQuery is the MANDATORY hook called whenever we
	// receive a query for the given domain.
	OnQuery func(domain string) DNSAction

	// onTimeout is the OPTIONAL channel where we emit a true
	// value each time there's a timeout. If you set this value
	// to a non-nil channel, then you MUST drain the channel
	// for each expected timeout. Otherwise, the code will just
	// ignore this field and nothing will be emitted.
	onTimeout chan bool
}

// DNSListener is the interface returned by DNSServer.Start.
type DNSListener interface {
	io.Closer
	LocalAddr() net.Addr
}

// Start starts this server.
func (p *DNSServer) Start(address string) (DNSListener, error) {
	pconn, _, err := p.start(address)
	return pconn, err
}

func (p *DNSServer) start(address string) (DNSListener, <-chan interface{}, error) {
	pconn, err := net.ListenPacket("udp", address)
	if err != nil {
		return nil, nil, err
	}
	done := make(chan interface{})
	go p.mainloop(pconn, done)
	return pconn, done, nil
}

func (p *DNSServer) mainloop(pconn net.PacketConn, done chan<- interface{}) {
	defer close(done)
	for p.oneloop(pconn) {
		// nothing
	}
}

func (p *DNSServer) oneloop(pconn net.PacketConn) bool {
	buffer := make([]byte, 1<<17)
	count, addr, err := pconn.ReadFrom(buffer)
	if err != nil {
		return !strings.HasSuffix(err.Error(), "use of closed network connection")
	}
	buffer = buffer[:count]
	go p.serveAsync(pconn, addr, buffer)
	return true
}

func (p *DNSServer) emit(pconn net.PacketConn, addr net.Addr, reply ...*dns.Msg) (success int) {
	for _, entry := range reply {
		replyBytes, err := entry.Pack()
		if err != nil {
			continue
		}
		pconn.WriteTo(replyBytes, addr)
		success++ // we use this value in tests
	}
	return
}

func (p *DNSServer) serveAsync(pconn net.PacketConn, addr net.Addr, buffer []byte) {
	query := &dns.Msg{}
	if err := query.Unpack(buffer); err != nil {
		return
	}
	if len(query.Question) < 1 {
		return // just discard the query
	}
	name := query.Question[0].Name
	switch p.OnQuery(name) {
	case DNSActionNXDOMAIN:
		p.emit(pconn, addr, p.nxdomain(query))
	case DNSActionLocalHost:
		p.emit(pconn, addr, p.localHost(query))
	case DNSActionNoAnswer:
		p.emit(pconn, addr, p.empty(query))
	case DNSActionTimeout:
		if p.onTimeout != nil {
			p.onTimeout <- true
		}
	case DNSActionCache:
		p.emit(pconn, addr, p.cache(name, query))
	case DNSActionLocalHostPlusCache:
		p.emit(pconn, addr, p.localHost(query))
		time.Sleep(10 * time.Millisecond)
		p.emit(pconn, addr, p.cache(name, query))
	default:
		p.emit(pconn, addr, p.refused(query))
	}
}

func (p *DNSServer) refused(query *dns.Msg) *dns.Msg {
	m := new(dns.Msg)
	m.SetRcode(query, dns.RcodeRefused)
	return m
}

func (p *DNSServer) nxdomain(query *dns.Msg) *dns.Msg {
	m := new(dns.Msg)
	m.SetRcode(query, dns.RcodeNameError)
	return m
}

func (p *DNSServer) localHost(query *dns.Msg) *dns.Msg {
	return p.compose(query, net.IPv6loopback, net.IPv4(127, 0, 0, 1))
}

func (p *DNSServer) empty(query *dns.Msg) *dns.Msg {
	return p.compose(query)
}

func (p *DNSServer) compose(query *dns.Msg, ips ...net.IP) *dns.Msg {
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

func (p *DNSServer) cache(name string, query *dns.Msg) *dns.Msg {
	addrs := p.Cache[name]
	var ipAddrs []net.IP
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil {
			ipAddrs = append(ipAddrs, ip)
		}
	}
	if len(ipAddrs) <= 0 {
		return p.nxdomain(query)
	}
	return p.compose(query, ipAddrs...)
}
