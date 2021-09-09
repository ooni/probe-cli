// Package resolver contains a censoring DNS resolver. Most queries are
// answered without censorship, but selected queries could either be
// discarded or replied to with a bogon or NXDOMAIN answer.
package resolver

import (
	"context"
	"net"
	"strings"

	"github.com/miekg/dns"
)

// Resolver resolves domain names.
type Resolver interface {
	LookupHost(ctx context.Context, hostname string) ([]string, error)
}

// CensoringResolver is a censoring resolver.
type CensoringResolver struct {
	blocked    []string
	hijacked   []string
	ignored    []string
	lookupHost func(ctx context.Context, host string) ([]string, error)
}

// NewCensoringResolver creates a new CensoringResolver instance using
// the specified list of keywords to censor. blocked is the list of
// keywords that trigger NXDOMAIN if they appear in a query. hijacked
// is similar but redirects to 127.0.0.1, where the transparent HTTP
// and TLS proxies will pick them up. dnsNetwork and dnsAddress are the
// settings to configure the upstream, non censored DNS.
func NewCensoringResolver(
	blocked, hijacked, ignored []string, uncensored Resolver,
) *CensoringResolver {
	return &CensoringResolver{
		blocked:    blocked,
		hijacked:   hijacked,
		ignored:    ignored,
		lookupHost: uncensored.LookupHost,
	}
}

func (r *CensoringResolver) roundtrip(rw dns.ResponseWriter, req *dns.Msg) {
	name := req.Question[0].Name
	addrs, err := r.lookupHost(context.Background(), name)
	var ips []net.IP
	if err == nil {
		for _, addr := range addrs {
			if ip := net.ParseIP(addr); ip != nil {
				ips = append(ips, ip)
			}
		}
	}
	r.reply(rw, req, ips)
}

func (r *CensoringResolver) reply(
	rw dns.ResponseWriter, req *dns.Msg, ips []net.IP,
) {
	m := new(dns.Msg)
	m.Compress = true
	m.MsgHdr.RecursionAvailable = true
	m.SetReply(req)
	for _, ip := range ips {
		ipv6 := strings.Contains(ip.String(), ":")
		if !ipv6 && req.Question[0].Qtype == dns.TypeA {
			m.Answer = append(m.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   req.Question[0].Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				A: ip,
			})
		}
	}
	if m.Answer == nil {
		m.SetRcode(req, dns.RcodeNameError)
	}
	rw.WriteMsg(m)
}

func (r *CensoringResolver) failure(rw dns.ResponseWriter, req *dns.Msg) {
	m := new(dns.Msg)
	m.Compress = true
	m.MsgHdr.RecursionAvailable = true
	m.SetRcode(req, dns.RcodeServerFailure)
	rw.WriteMsg(m)
}

// ServeDNS serves a DNS request
func (r *CensoringResolver) ServeDNS(rw dns.ResponseWriter, req *dns.Msg) {
	if len(req.Question) < 1 {
		r.failure(rw, req)
		return
	}
	name := req.Question[0].Name
	for _, pattern := range r.blocked {
		if strings.Contains(name, pattern) {
			r.reply(rw, req, nil)
			return
		}
	}
	for _, pattern := range r.hijacked {
		if strings.Contains(name, pattern) {
			r.reply(rw, req, []net.IP{net.IPv4(127, 0, 0, 1)})
			return
		}
	}
	for _, pattern := range r.ignored {
		if strings.Contains(name, pattern) {
			return
		}
	}
	r.roundtrip(rw, req)
}

// Start starts the DNS resolver
func (r *CensoringResolver) Start(address string) (*dns.Server, error) {
	packetconn, err := net.ListenPacket("udp", address)
	if err != nil {
		return nil, err
	}
	server := &dns.Server{
		Addr:       address,
		Handler:    r,
		Net:        "udp",
		PacketConn: packetconn,
	}
	go server.ActivateAndServe()
	return server, nil
}
