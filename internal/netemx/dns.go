package netemx

import (
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type DoHServer struct {
	rec map[string]net.IP
	mu  sync.Mutex
}

func (p *DoHServer) AddRecord(domain string, ip string) {
	defer p.mu.Unlock()
	p.mu.Lock()
	if p.rec == nil {
		p.rec = make(map[string]net.IP)
	}
	p.rec[domain+"."] = net.ParseIP(ip)
}

func (p *DoHServer) lookup(name string) (net.IP, bool) {
	defer p.mu.Unlock()
	p.mu.Lock()
	ip, found := p.rec[name]
	return ip, found
}

func (p *DoHServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer p.HTTPPanicToInternalServerError(w)
	rawQuery, err := io.ReadAll(r.Body)
	runtimex.PanicOnError(err, "io.ReadAll failed")
	query := &dns.Msg{}
	err = query.Unpack(rawQuery)
	runtimex.PanicOnError(err, "query.Unpack failed")
	runtimex.PanicIfTrue(query.Response, "is a response")

	ip, found := p.lookup(query.Question[0].Name)
	var response *dns.Msg
	if found {
		response = filtering.DNSComposeResponse(query, ip)
	} else {
		response = &dns.Msg{}
		response.SetRcode(query, dns.RcodeNameError)
	}
	rawResponse, err := response.Pack()
	runtimex.PanicOnError(err, "response.Pack failed")
	w.Header().Add("content-type", "application/dns-message")
	w.Write(rawResponse)
}

func (p *DoHServer) HTTPPanicToInternalServerError(w http.ResponseWriter) {
	if r := recover(); r != nil {
		w.WriteHeader(500)
	}
}
