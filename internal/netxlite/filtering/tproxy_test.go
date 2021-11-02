package filtering

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
)

func TestNewTProxyConfig(t *testing.T) {
	t.Run("with nonexistent file", func(t *testing.T) {
		config, err := NewTProxyConfig(filepath.Join("testdata", "nonexistent"))
		if !errors.Is(err, syscall.ENOENT) {
			t.Fatal("unexpected err", err)
		}
		if config != nil {
			t.Fatal("expected nil config here")
		}
	})

	t.Run("with file containing invalid JSON", func(t *testing.T) {
		config, err := NewTProxyConfig(filepath.Join("testdata", "invalid.json"))
		if err == nil || !strings.HasSuffix(err.Error(), "unexpected end of JSON input") {
			t.Fatal("unexpected err", err)
		}
		if config != nil {
			t.Fatal("expected nil config here")
		}
	})

	t.Run("with file containing valid JSON", func(t *testing.T) {
		config, err := NewTProxyConfig(filepath.Join("testdata", "valid.json"))
		if err != nil {
			t.Fatal(err)
		}
		if config == nil {
			t.Fatal("expected non-nil config here")
		}
		if config.Domains["x.org."] != "pass" {
			t.Fatal("did not auto-canonicalize names")
		}
	})
}

func TestNewTProxy(t *testing.T) {
	t.Run("successful creation and destruction", func(t *testing.T) {
		config := &TProxyConfig{}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		if err := proxy.Close(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("cannot create DNS listener", func(t *testing.T) {
		config := &TProxyConfig{}
		proxy, err := newTProxy(config, log.Log, "127.0.0.1", "", "")
		if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
			t.Fatal("unexpected err", err)
		}
		if proxy != nil {
			t.Fatal("expected nil proxy here")
		}
	})

	t.Run("cannot create TLS listener", func(t *testing.T) {
		config := &TProxyConfig{}
		proxy, err := newTProxy(config, log.Log, "127.0.0.1:0", "127.0.0.1", "")
		if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
			t.Fatal("unexpected err", err)
		}
		if proxy != nil {
			t.Fatal("expected nil proxy here")
		}
	})

	t.Run("cannot create HTTP listener", func(t *testing.T) {
		config := &TProxyConfig{}
		proxy, err := newTProxy(config, log.Log, "127.0.0.1:0", "127.0.0.1:0", "127.0.0.1")
		if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
			t.Fatal("unexpected err", err)
		}
		if proxy != nil {
			t.Fatal("expected nil proxy here")
		}
	})
}

func TestTProxyQUIC(t *testing.T) {
	t.Run("ListenUDP", func(t *testing.T) {
		t.Run("failure", func(t *testing.T) {
			proxy, err := NewTProxy(&TProxyConfig{}, log.Log)
			if err != nil {
				t.Fatal(err)
			}
			defer proxy.Close()
			pconn, err := proxy.ListenUDP("tcp", &net.UDPAddr{})
			if err == nil || !strings.HasSuffix(err.Error(), "unknown network tcp") {
				t.Fatal("unexpected err", err)
			}
			if pconn != nil {
				t.Fatal("expected nil pconn here")
			}
		})

		t.Run("success", func(t *testing.T) {
			proxy, err := NewTProxy(&TProxyConfig{}, log.Log)
			if err != nil {
				t.Fatal(err)
			}
			defer proxy.Close()
			pconn, err := proxy.ListenUDP("udp", &net.UDPAddr{})
			if err != nil {
				t.Fatal(err)
			}
			uconn := pconn.(*tProxyUDPLikeConn)
			if uconn.proxy != proxy {
				t.Fatal("proxy not correctly set")
			}
			if _, okay := uconn.UDPLikeConn.(*net.UDPConn); !okay {
				t.Fatal("underlying connection should be an UDPConn")
			}
			uconn.Close()
		})
	})

	t.Run("WriteTo", func(t *testing.T) {
		t.Run("without the drop policy", func(t *testing.T) {
			proxy, err := NewTProxy(&TProxyConfig{}, log.Log)
			if err != nil {
				t.Fatal(err)
			}
			defer proxy.Close()
			var called bool
			proxy.listenUDP = func(network string, laddr *net.UDPAddr) (quicx.UDPLikeConn, error) {
				return &mocks.QUICUDPLikeConn{
					MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
						called = true
						return len(p), nil
					},
				}, nil
			}
			pconn, err := proxy.ListenUDP("udp", &net.UDPAddr{})
			if err != nil {
				t.Fatal(err)
			}
			data := make([]byte, 128)
			count, err := pconn.WriteTo(data, &net.UDPAddr{})
			if err != nil {
				t.Fatal(err)
			}
			if count != len(data) {
				t.Fatal("unexpected number of bytes written")
			}
			if !called {
				t.Fatal("not called")
			}
		})

		t.Run("with the drop policy", func(t *testing.T) {
			config := &TProxyConfig{
				Endpoints: map[string]TProxyPolicy{
					"127.0.0.1:1234/udp": TProxyPolicyDropData,
				},
			}
			proxy, err := NewTProxy(config, log.Log)
			if err != nil {
				t.Fatal(err)
			}
			defer proxy.Close()
			var called bool
			proxy.listenUDP = func(network string, laddr *net.UDPAddr) (quicx.UDPLikeConn, error) {
				return &mocks.QUICUDPLikeConn{
					MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
						called = true
						return len(p), nil
					},
				}, nil
			}
			pconn, err := proxy.ListenUDP("udp", &net.UDPAddr{})
			if err != nil {
				t.Fatal(err)
			}
			data := make([]byte, 128)
			destAddr := &net.UDPAddr{
				IP:   net.IPv4(127, 0, 0, 1),
				Port: 1234,
				Zone: "",
			}
			count, err := pconn.WriteTo(data, destAddr)
			if err != nil {
				t.Fatal(err)
			}
			if count != len(data) {
				t.Fatal("unexpected number of bytes written")
			}
			if called {
				t.Fatal("called")
			}
		})
	})
}

func TestTProxyLookupHost(t *testing.T) {
	t.Run("without filtering", func(t *testing.T) {
		config := &TProxyConfig{}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		defer proxy.Close()
		ctx := context.Background()
		addrs, err := proxy.LookupHost(ctx, "dns.google")
		if err != nil {
			t.Fatal(err)
		}
		if len(addrs) < 2 {
			t.Fatal("too few addrs")
		}
	})

	t.Run("with filtering", func(t *testing.T) {
		config := &TProxyConfig{
			Domains: map[string]DNSAction{
				"dns.google.": DNSActionNXDOMAIN,
			},
		}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		defer proxy.Close()
		ctx := context.Background()
		addrs, err := proxy.LookupHost(ctx, "dns.google")
		if err == nil || err.Error() != "dns_nxdomain_error" {
			t.Fatal("unexpected err", err)
		}
		if len(addrs) != 0 {
			t.Fatal("too many addrs")
		}
	})
}

func TestTProxyOnIncomingSNI(t *testing.T) {
	t.Run("without filtering", func(t *testing.T) {
		config := &TProxyConfig{
			Endpoints: map[string]TProxyPolicy{
				"8.8.8.8:443/tcp": TProxyPolicyHijackTLS,
			},
		}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		defer proxy.Close()
		ctx := context.Background()
		dialer := proxy.NewTProxyDialer(10 * time.Second)
		conn, err := dialer.DialContext(ctx, "tcp", "8.8.8.8:443")
		if err != nil {
			t.Fatal(err)
		}
		tconn := tls.Client(conn, &tls.Config{ServerName: "dns.google"})
		err = tconn.HandshakeContext(ctx)
		if err != nil {
			t.Fatal(err)
		}
		tconn.Close()
	})

	t.Run("with filtering", func(t *testing.T) {
		config := &TProxyConfig{
			Endpoints: map[string]TProxyPolicy{
				"8.8.8.8:443/tcp": TProxyPolicyHijackTLS,
			},
			SNIs: map[string]TLSAction{
				"dns.google": TLSActionReset,
			},
		}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		defer proxy.Close()
		ctx := context.Background()
		dialer := proxy.NewTProxyDialer(10 * time.Second)
		conn, err := dialer.DialContext(ctx, "tcp", "8.8.8.8:443")
		if err != nil {
			t.Fatal(err)
		}
		tlsh := netxlite.NewTLSHandshakerStdlib(log.Log)
		tconn, _, err := tlsh.Handshake(ctx, conn, &tls.Config{ServerName: "dns.google"})
		if err == nil || err.Error() != netxlite.FailureConnectionReset {
			t.Fatal("unexpected err", err)
		}
		if tconn != nil {
			t.Fatal("expected nil tconn")
		}
		conn.Close()
	})
}

func TestTProxyOnIncomingHost(t *testing.T) {
	t.Run("without filtering", func(t *testing.T) {
		config := &TProxyConfig{
			Endpoints: map[string]TProxyPolicy{
				"130.192.16.171:80/tcp": TProxyPolicyHijackHTTP,
			},
		}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		defer proxy.Close()
		dialer := proxy.NewTProxyDialer(10 * time.Second)
		req, err := http.NewRequest("GET", "http://130.192.16.171:80", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "nexa.polito.it"
		txp := &http.Transport{DialContext: dialer.DialContext}
		resp, err := txp.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	})

	t.Run("with filtering", func(t *testing.T) {
		config := &TProxyConfig{
			Endpoints: map[string]TProxyPolicy{
				"130.192.16.171:80/tcp": TProxyPolicyHijackHTTP,
			},
			Hosts: map[string]HTTPAction{
				"nexa.polito.it": HTTPActionReset,
			},
		}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		defer proxy.Close()
		dialer := netxlite.WrapDialer(
			log.Log,
			netxlite.NewResolverStdlib(log.Log),
			&tProxyDialerAdapter{
				proxy.NewTProxyDialer(10 * time.Second),
			},
		)
		req, err := http.NewRequest("GET", "http://130.192.16.171:80", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "nexa.polito.it"
		txp := &http.Transport{DialContext: dialer.DialContext}
		resp, err := txp.RoundTrip(req)
		if err == nil || !strings.HasSuffix(err.Error(), netxlite.FailureConnectionReset) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("expected nil resp here")
		}
	})
}

func TestTProxyDial(t *testing.T) {
	t.Run("with drop SYN", func(t *testing.T) {
		config := &TProxyConfig{
			Endpoints: map[string]TProxyPolicy{
				"130.192.16.171:80/tcp": TProxyPolicyTCPDropSYN,
			},
		}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		defer proxy.Close()
		dialer := proxy.NewTProxyDialer(10 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, "GET", "http://130.192.16.171:80", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "nexa.polito.it"
		txp := &http.Transport{DialContext: dialer.DialContext}
		resp, err := txp.RoundTrip(req)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("expected nil resp here")
		}
	})

	t.Run("with reject", func(t *testing.T) {
		config := &TProxyConfig{
			Endpoints: map[string]TProxyPolicy{
				"130.192.16.171:80/tcp": TProxyPolicyTCPReject,
			},
		}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		defer proxy.Close()
		dialer := netxlite.WrapDialer(log.Log,
			netxlite.NewResolverStdlib(log.Log),
			&tProxyDialerAdapter{
				proxy.NewTProxyDialer(10 * time.Second)})
		req, err := http.NewRequest("GET", "http://130.192.16.171:80", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "nexa.polito.it"
		txp := &http.Transport{DialContext: dialer.DialContext}
		resp, err := txp.RoundTrip(req)
		if err == nil || !strings.HasSuffix(err.Error(), netxlite.FailureConnectionRefused) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("expected nil resp here")
		}
	})

	t.Run("with drop data", func(t *testing.T) {
		config := &TProxyConfig{
			Endpoints: map[string]TProxyPolicy{
				"130.192.16.171:80/tcp": TProxyPolicyDropData,
			},
		}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		defer proxy.Close()
		dialer := proxy.NewTProxyDialer(10 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(
			ctx, "GET", "http://130.192.16.171:80", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "nexa.polito.it"
		txp := &http.Transport{DialContext: dialer.DialContext}
		resp, err := txp.RoundTrip(req)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("expected nil resp here")
		}
	})

	t.Run("with hijack DNS", func(t *testing.T) {
		config := &TProxyConfig{
			Endpoints: map[string]TProxyPolicy{
				"8.8.8.8:53/udp": TProxyPolicyHijackDNS,
			},
			Domains: map[string]DNSAction{
				"example.com.": DNSActionNXDOMAIN,
			},
		}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		defer proxy.Close()
		dialer := proxy.NewTProxyDialer(10 * time.Second)
		resolver := netxlite.NewResolverUDP(
			log.Log, &tProxyDialerAdapter{dialer}, "8.8.8.8:53")
		addrs, err := resolver.LookupHost(context.Background(), "example.com")
		if err == nil || err.Error() != netxlite.FailureDNSNXDOMAINError {
			t.Fatal("unexpected err", err)
		}
		if len(addrs) != 0 {
			t.Fatal("expected no addrs here")
		}
	})

	t.Run("with invalid destination address", func(t *testing.T) {
		config := &TProxyConfig{}
		proxy, err := NewTProxy(config, log.Log)
		if err != nil {
			t.Fatal(err)
		}
		defer proxy.Close()
		dialer := proxy.NewTProxyDialer(10 * time.Second)
		ctx := context.Background()
		conn, err := dialer.DialContext(ctx, "tcp", "127.0.0.1")
		if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn here")
		}
	})
}
