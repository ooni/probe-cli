package filtering

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDNSProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	newProxyWithCache := func(action DNSAction, cache map[string][]string) (DNSListener, <-chan interface{}, error) {
		p := &DNSProxy{
			Cache: cache,
			OnQuery: func(domain string) DNSAction {
				return action
			},
		}
		return p.start("127.0.0.1:0")
	}

	newProxy := func(action DNSAction) (DNSListener, <-chan interface{}, error) {
		return newProxyWithCache(action, nil)
	}

	newresolver := func(listener DNSListener) model.Resolver {
		dlr := netxlite.NewDialerWithoutResolver(log.Log)
		r := netxlite.NewResolverUDP(log.Log, dlr, listener.LocalAddr().String())
		return r
	}

	t.Run("DNSActionPass", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newProxy(DNSActionPass)
		if err != nil {
			t.Fatal(err)
		}
		r := newresolver(listener)
		addrs, err := r.LookupHost(ctx, "dns.google")
		if err != nil {
			t.Fatal(err)
		}
		if addrs == nil {
			t.Fatal("unexpected empty addrs")
		}
		var found bool
		for _, addr := range addrs {
			found = found || addr == "8.8.8.8"
		}
		if !found {
			t.Fatal("did not find 8.8.8.8")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionNXDOMAIN", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newProxy(DNSActionNXDOMAIN)
		if err != nil {
			t.Fatal(err)
		}
		r := newresolver(listener)
		addrs, err := r.LookupHost(ctx, "dns.google")
		if err == nil || err.Error() != netxlite.FailureDNSNXDOMAINError {
			t.Fatal("unexpected err", err)
		}
		if addrs != nil {
			t.Fatal("expected empty addrs")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionRefused", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newProxy(DNSActionRefused)
		if err != nil {
			t.Fatal(err)
		}
		r := newresolver(listener)
		addrs, err := r.LookupHost(ctx, "dns.google")
		if err == nil || err.Error() != netxlite.FailureDNSRefusedError {
			t.Fatal("unexpected err", err)
		}
		if addrs != nil {
			t.Fatal("expected empty addrs")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionLocalHost", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newProxy(DNSActionLocalHost)
		if err != nil {
			t.Fatal(err)
		}
		r := newresolver(listener)
		addrs, err := r.LookupHost(ctx, "dns.google")
		if err != nil {
			t.Fatal(err)
		}
		if addrs == nil {
			t.Fatal("expected non-empty addrs")
		}
		var found bool
		for _, addr := range addrs {
			found = found || addr == "127.0.0.1"
		}
		if !found {
			t.Fatal("did not find 127.0.0.1")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionEmpty", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newProxy(DNSActionNoAnswer)
		if err != nil {
			t.Fatal(err)
		}
		r := newresolver(listener)
		addrs, err := r.LookupHost(ctx, "dns.google")
		if err == nil || err.Error() != netxlite.FailureDNSNoAnswer {
			t.Fatal("unexpected err", err)
		}
		if addrs != nil {
			t.Fatal("expected empty addrs")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionTimeout", func(t *testing.T) {
		// Implementation note: if you see this test running for more
		// than one second, then it means we're not checking the context
		// immediately. We should be improving there but we need to be
		// careful because lots of legacy code uses SerialResolver.
		const timeout = time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		listener, done, err := newProxy(DNSActionTimeout)
		if err != nil {
			t.Fatal(err)
		}
		r := newresolver(listener)
		addrs, err := r.LookupHost(ctx, "dns.google")
		if err == nil || err.Error() != netxlite.FailureGenericTimeoutError {
			t.Fatal("unexpected err", err)
		}
		if addrs != nil {
			t.Fatal("expected empty addrs")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionCache without entries", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newProxyWithCache(DNSActionCache, nil)
		if err != nil {
			t.Fatal(err)
		}
		r := newresolver(listener)
		addrs, err := r.LookupHost(ctx, "dns.google")
		if err == nil || err.Error() != netxlite.FailureDNSNXDOMAINError {
			t.Fatal("unexpected err", err)
		}
		if addrs != nil {
			t.Fatal("expected empty addrs")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("DNSActionCache with entries", func(t *testing.T) {
		ctx := context.Background()
		cache := map[string][]string{
			"dns.google.": {"8.8.8.8", "8.8.4.4"},
		}
		listener, done, err := newProxyWithCache(DNSActionCache, cache)
		if err != nil {
			t.Fatal(err)
		}
		r := newresolver(listener)
		addrs, err := r.LookupHost(ctx, "dns.google")
		if err != nil {
			t.Fatal(err)
		}
		if len(addrs) != 2 {
			t.Fatal("expected two entries")
		}
		if addrs[0] != "8.8.8.8" {
			t.Fatal("invalid first entry")
		}
		if addrs[1] != "8.8.4.4" {
			t.Fatal("invalid second entry")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("Start with invalid address", func(t *testing.T) {
		p := &DNSProxy{}
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
			p := &DNSProxy{}
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
			p := &DNSProxy{}
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
			p := &DNSProxy{}
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

		t.Run("reply fails", func(t *testing.T) {
			p := &DNSProxy{}
			conn := &mocks.UDPLikeConn{
				MockReadFrom: func(p []byte) (n int, addr net.Addr, err error) {
					query := &dns.Msg{}
					query.Question = append(query.Question, dns.Question{})
					query.Question = append(query.Question, dns.Question{})
					data, err := query.Pack()
					if err != nil {
						panic(err)
					}
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

		t.Run("pack fails", func(t *testing.T) {
			p := &DNSProxy{
				mockableReply: func(query *dns.Msg) (*dns.Msg, error) {
					reply := &dns.Msg{}
					reply.MsgHdr.Rcode = -1 // causes pack to fail
					return reply, nil
				},
			}
			conn := &mocks.UDPLikeConn{
				MockReadFrom: func(p []byte) (n int, addr net.Addr, err error) {
					query := &dns.Msg{}
					query.Question = append(query.Question, dns.Question{})
					data, err := query.Pack()
					if err != nil {
						panic(err)
					}
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

	t.Run("proxy", func(t *testing.T) {
		t.Run("with response", func(t *testing.T) {
			p := &DNSProxy{}
			query := &dns.Msg{}
			query.Response = true
			reply, err := p.proxy(query)
			if !errors.Is(err, errDNSExpectedQueryNotResponse) {
				t.Fatal("unexpected err", err)
			}
			if reply != nil {
				t.Fatal("expected nil reply")
			}
		})

		t.Run("with no questions", func(t *testing.T) {
			p := &DNSProxy{}
			query := &dns.Msg{}
			reply, err := p.proxy(query)
			if !errors.Is(err, errDNSExpectedSingleQuestion) {
				t.Fatal("unexpected err", err)
			}
			if reply != nil {
				t.Fatal("expected nil reply")
			}
		})

		t.Run("round trip fails", func(t *testing.T) {
			p := &DNSProxy{
				UpstreamEndpoint: "antani",
			}
			query := &dns.Msg{}
			query.Question = append(query.Question, dns.Question{})
			reply, err := p.proxy(query)
			if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
				t.Fatal("unexpected err", err)
			}
			if reply != nil {
				t.Fatal("expected nil reply here")
			}
		})
	})
}
