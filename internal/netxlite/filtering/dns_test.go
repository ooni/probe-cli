package filtering

import (
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/randx"
)

func TestDNSServer(t *testing.T) {
	newServerWithCache := func(action DNSAction, cache map[string][]string) (
		*DNSServer, DNSListener, <-chan interface{}, error) {
		p := &DNSServer{
			Cache: cache,
			OnQuery: func(domain string) DNSAction {
				return action
			},
		}
		listener, done, err := p.start("127.0.0.1:0")
		return p, listener, done, err
	}

	newServer := func(action DNSAction) (*DNSServer, DNSListener, <-chan interface{}, error) {
		return newServerWithCache(action, nil)
	}

	newQuery := func(qtype uint16) *dns.Msg {
		question := dns.Question{
			Name:   dns.Fqdn("dns.google"),
			Qtype:  qtype,
			Qclass: dns.ClassINET,
		}
		query := new(dns.Msg)
		query.Id = dns.Id()
		query.RecursionDesired = true
		query.Question = make([]dns.Question, 1)
		query.Question[0] = question
		return query
	}

	t.Run("DNSActionNXDOMAIN", func(t *testing.T) {
		_, listener, done, err := newServer(DNSActionNXDOMAIN)
		if err != nil {
			t.Fatal(err)
		}
		reply, err := dns.Exchange(newQuery(dns.TypeA), listener.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}
		if reply.Rcode != dns.RcodeNameError {
			t.Fatal("unexpected rcode")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionRefused", func(t *testing.T) {
		_, listener, done, err := newServer(DNSActionRefused)
		if err != nil {
			t.Fatal(err)
		}
		reply, err := dns.Exchange(newQuery(dns.TypeA), listener.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}
		if reply.Rcode != dns.RcodeRefused {
			t.Fatal("unexpected rcode")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionLocalHost", func(t *testing.T) {
		_, listener, done, err := newServer(DNSActionLocalHost)
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		reply, err := dns.Exchange(newQuery(dns.TypeA), listener.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}
		if reply.Rcode != dns.RcodeSuccess {
			t.Fatal("unexpected rcode")
		}
		var found bool
		for _, ans := range reply.Answer {
			switch v := ans.(type) {
			case *dns.A:
				found = found || v.A.String() == "127.0.0.1"
			}
		}
		if !found {
			t.Fatal("did not find 127.0.0.1")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionEmpty", func(t *testing.T) {
		_, listener, done, err := newServer(DNSActionNoAnswer)
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		reply, err := dns.Exchange(newQuery(dns.TypeA), listener.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}
		if reply.Rcode != dns.RcodeSuccess {
			t.Fatal("unexpected rcode")
		}
		if len(reply.Answer) != 0 {
			t.Fatal("expected no answers")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionTimeout", func(t *testing.T) {
		srvr, listener, done, err := newServer(DNSActionTimeout)
		if err != nil {
			t.Fatal(err)
		}
		c := &dns.Client{}
		conn, err := c.Dial(listener.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}
		srvr.onTimeout = func() {
			conn.Close() // forces the exchange to interrupt ~immediately
		}
		reply, _, err := c.ExchangeWithConn(newQuery(dns.TypeA), conn)
		if !errors.Is(err, net.ErrClosed) {
			t.Fatal("unexpected err", err)
		}
		if reply != nil {
			t.Fatal("expected nil reply here")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionCache without entries", func(t *testing.T) {
		_, listener, done, err := newServerWithCache(DNSActionCache, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		reply, err := dns.Exchange(newQuery(dns.TypeA), listener.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}
		if reply.Rcode != dns.RcodeNameError {
			t.Fatal("unexpected rcode")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionCache with IPv4 entry", func(t *testing.T) {
		cache := map[string][]string{
			"dns.google.": {"8.8.8.8"},
		}
		_, listener, done, err := newServerWithCache(DNSActionCache, cache)
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		reply, err := dns.Exchange(newQuery(dns.TypeA), listener.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}
		if reply.Rcode != dns.RcodeSuccess {
			t.Fatal("unexpected rcode")
		}
		var found bool
		for _, ans := range reply.Answer {
			switch v := ans.(type) {
			case *dns.A:
				found = found || v.A.String() == "8.8.8.8"
			}
		}
		if !found {
			t.Fatal("did not find 8.8.8.8")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionCache with IPv6 entry", func(t *testing.T) {
		cache := map[string][]string{
			"dns.google.": {"2001:4860:4860::8888"},
		}
		_, listener, done, err := newServerWithCache(DNSActionCache, cache)
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		reply, err := dns.Exchange(newQuery(dns.TypeAAAA), listener.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}
		if reply.Rcode != dns.RcodeSuccess {
			t.Fatal("unexpected rcode")
		}
		var found bool
		for _, ans := range reply.Answer {
			switch v := ans.(type) {
			case *dns.AAAA:
				found = found || v.AAAA.String() == "2001:4860:4860::8888"
			}
		}
		if !found {
			t.Fatal("did not find 2001:4860:4860::8888")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionLocalHostPlusCache", func(t *testing.T) {
		cache := map[string][]string{
			"dns.google.": {"2001:4860:4860::8888"},
		}
		_, listener, done, err := newServerWithCache(DNSActionLocalHostPlusCache, cache)
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		reply, err := dns.Exchange(newQuery(dns.TypeAAAA), listener.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}
		if reply.Rcode != dns.RcodeSuccess {
			t.Fatal("unexpected rcode")
		}
		var found bool
		for _, ans := range reply.Answer {
			switch v := ans.(type) {
			case *dns.AAAA:
				found = found || v.AAAA.String() == "::1"
			}
		}
		if !found {
			t.Fatal("did not find ::1")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("Start with invalid address", func(t *testing.T) {
		p := &DNSServer{}
		listener, err := p.Start("127.0.0.1")
		if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
			t.Fatal("unexpected err", err)
		}
		if listener != nil {
			t.Fatal("expected nil listener")
		}
	})

	t.Run("oneloop", func(t *testing.T) {
		t.Run("ReadFrom failure after which we should continue", func(t *testing.T) {
			expected := errors.New("mocked error")
			p := &DNSServer{}
			conn := &mocks.UDPLikeConn{
				MockReadFrom: func(p []byte) (n int, addr net.Addr, err error) {
					return 0, nil, expected
				},
			}
			okay := p.oneloop(conn)
			if !okay {
				t.Fatal("we should be okay after this error")
			}
		})

		t.Run("ReadFrom the connection is closed", func(t *testing.T) {
			expected := errors.New("use of closed network connection")
			p := &DNSServer{}
			conn := &mocks.UDPLikeConn{
				MockReadFrom: func(p []byte) (n int, addr net.Addr, err error) {
					return 0, nil, expected
				},
			}
			okay := p.oneloop(conn)
			if okay {
				t.Fatal("we should not be okay after this error")
			}
		})

		t.Run("Unpack fails", func(t *testing.T) {
			p := &DNSServer{}
			conn := &mocks.UDPLikeConn{
				MockReadFrom: func(p []byte) (n int, addr net.Addr, err error) {
					if len(p) < 4 {
						panic("buffer too small")
					}
					p[0] = 7
					return 1, &net.UDPAddr{}, nil
				},
			}
			okay := p.oneloop(conn)
			if !okay {
				t.Fatal("we should be okay after this error")
			}
		})

		t.Run("no questions", func(t *testing.T) {
			query := newQuery(dns.TypeA)
			query.Question = nil // remove the question
			data, err := query.Pack()
			if err != nil {
				t.Fatal(err)
			}
			p := &DNSServer{}
			conn := &mocks.UDPLikeConn{
				MockReadFrom: func(p []byte) (n int, addr net.Addr, err error) {
					if len(p) < len(data) {
						panic("buffer too small")
					}
					copy(p, data)
					return len(data), &net.UDPAddr{}, nil
				},
			}
			okay := p.oneloop(conn)
			if !okay {
				t.Fatal("we should be okay after this error")
			}
		})
	})

	t.Run("pack fails", func(t *testing.T) {
		query := newQuery(dns.TypeA)
		query.Question[0].Name = randx.Letters(1024) // should be too large
		p := &DNSServer{}
		count := p.emit(&mocks.UDPLikeConn{}, &mocks.Addr{}, query)
		if count != 0 {
			t.Fatal("expected to see zero here")
		}
	})
}
