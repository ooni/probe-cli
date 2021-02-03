package resolver

import (
	"strings"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/uncensored"
)

func TestPass(t *testing.T) {
	server := newresolver(t, []string{"ooni.io"}, []string{"ooni.nu"}, nil)
	checkrequest(t, server, "example.com", "success", nil)
	killserver(t, server)
}

func TestBlock(t *testing.T) {
	server := newresolver(t, []string{"ooni.io"}, []string{"ooni.nu"}, nil)
	checkrequest(t, server, "mia-ps.ooni.io", "blocked", nil)
	killserver(t, server)
}

func TestRedirect(t *testing.T) {
	server := newresolver(t, []string{"ooni.io"}, []string{"ooni.nu"}, nil)
	checkrequest(t, server, "hkgmetadb.ooni.nu", "hijacked", nil)
	killserver(t, server)
}

func TestIgnore(t *testing.T) {
	server := newresolver(t, nil, nil, []string{"ooni.nu"})
	iotimeout := "i/o timeout"
	checkrequest(t, server, "hkgmetadb.ooni.nu", "hijacked", &iotimeout)
	killserver(t, server)
}

func TestLookupFailure(t *testing.T) {
	server := newresolver(t, nil, nil, nil)
	// we should receive same response as when we're blocked
	checkrequest(t, server, "example.antani", "blocked", nil)
	killserver(t, server)
}

func TestFailureNoQuestion(t *testing.T) {
	resolver := NewCensoringResolver(
		nil, nil, nil, uncensored.DefaultClient,
	)
	resolver.ServeDNS(&fakeResponseWriter{t: t}, new(dns.Msg))
}

func TestListenFailure(t *testing.T) {
	resolver := NewCensoringResolver(
		nil, nil, nil, uncensored.DefaultClient,
	)
	server, err := resolver.Start("8.8.8.8:53")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if server != nil {
		t.Fatal("expected nil server here")
	}
}

func newresolver(t *testing.T, blocked, hijacked, ignored []string) *dns.Server {
	resolver := NewCensoringResolver(
		blocked, hijacked, ignored,
		// using faster dns because dot here causes miekg/dns's
		// dns.Exchange to timeout and I don't want more complexity
		uncensored.Must(uncensored.NewClient("system:///")),
	)
	server, err := resolver.Start("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func killserver(t *testing.T, server *dns.Server) {
	err := server.Shutdown()
	if err != nil {
		t.Fatal(err)
	}
}

func checkrequest(
	t *testing.T, server *dns.Server, host string, expectStatus string,
	expectErrorSuffix *string,
) {
	address := server.PacketConn.LocalAddr().String()
	query := newquery(host)
	reply, err := dns.Exchange(query, address)
	if err != nil {
		if expectErrorSuffix != nil &&
			strings.HasSuffix(err.Error(), *expectErrorSuffix) {
			return
		}
		t.Fatal(err)
	}
	switch expectStatus {
	case "success":
		checksuccess(t, reply)
	case "hijacked":
		checkhijacked(t, reply)
	case "blocked":
		checkblocked(t, reply)
	default:
		panic("unexpected value")
	}
}

func checksuccess(t *testing.T, reply *dns.Msg) {
	if reply.Rcode != dns.RcodeSuccess {
		t.Fatal("unexpected rcode")
	}
	if len(reply.Answer) < 1 {
		t.Fatal("too few answers")
	}
	for _, answer := range reply.Answer {
		if rr, ok := answer.(*dns.A); ok {
			if rr.A.String() == "127.0.0.1" {
				t.Fatal("unexpected hijacked response here")
			}
		}
	}
}

func checkhijacked(t *testing.T, reply *dns.Msg) {
	if reply.Rcode != dns.RcodeSuccess {
		t.Fatal("unexpected rcode")
	}
	if len(reply.Answer) < 1 {
		t.Fatal("too few answers")
	}
	for _, answer := range reply.Answer {
		if rr, ok := answer.(*dns.A); ok {
			if rr.A.String() != "127.0.0.1" {
				t.Fatal("unexpected non-hijacked response here")
			}
		}
	}
}

func checkblocked(t *testing.T, reply *dns.Msg) {
	if reply.Rcode != dns.RcodeNameError {
		t.Fatal("unexpected rcode")
	}
	if len(reply.Answer) >= 1 {
		t.Fatal("too many answers")
	}
}

func newquery(name string) *dns.Msg {
	query := new(dns.Msg)
	query.Id = dns.Id()
	query.RecursionDesired = true
	query.Question = append(query.Question, dns.Question{
		Name:   dns.Fqdn(name),
		Qclass: dns.ClassINET,
		Qtype:  dns.TypeA,
	})
	return query
}

type fakeResponseWriter struct {
	dns.ResponseWriter
	t *testing.T
}

func (rw *fakeResponseWriter) WriteMsg(m *dns.Msg) error {
	if m.Rcode != dns.RcodeServerFailure {
		rw.t.Fatal("unexpected rcode")
	}
	return nil
}
