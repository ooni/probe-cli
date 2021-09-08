package netxlite_test

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
)

func TestResolver(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	t.Run("works as intended", func(t *testing.T) {
		// TODO(bassosimone): this is actually an integration
		// test but how to test this case?
		r := netxlite.NewResolverStdlib(log.Log)
		defer r.CloseIdleConnections()
		addrs, err := r.LookupHost(context.Background(), "dns.google.com")
		if err != nil {
			t.Fatal(err)
		}
		if addrs == nil {
			t.Fatal("expected non-nil result here")
		}
	})
}

func TestHTTPTransport(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	t.Run("works as intended", func(t *testing.T) {
		d := netxlite.NewDialerWithResolver(log.Log, netxlite.NewResolverStdlib(log.Log))
		td := netxlite.NewTLSDialer(d, netxlite.NewTLSHandshakerStdlib(log.Log))
		txp := netxlite.NewHTTPTransport(log.Log, d, td)
		client := &http.Client{Transport: txp}
		resp, err := client.Get("https://www.google.com/robots.txt")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		client.CloseIdleConnections()
	})
}

func TestHTTP3Transport(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	t.Run("works as intended", func(t *testing.T) {
		d := netxlite.NewQUICDialerWithResolver(
			netxlite.NewQUICListener(),
			log.Log,
			netxlite.NewResolverStdlib(log.Log),
		)
		txp := netxlite.NewHTTP3Transport(log.Log, d, &tls.Config{})
		client := &http.Client{Transport: txp}
		resp, err := client.Get("https://www.google.com/robots.txt")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		txp.CloseIdleConnections()
	})
}

func TestUTLSHandshaker(t *testing.T) {
	t.Run("with chrome fingerprint", func(t *testing.T) {
		h := netxlite.NewTLSHandshakerUTLS(log.Log, &utls.HelloChrome_Auto)
		cfg := &tls.Config{ServerName: "google.com"}
		conn, err := net.Dial("tcp", "google.com:443")
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		conn, _, err = h.Handshake(context.Background(), conn, cfg)
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		if conn == nil {
			t.Fatal("nil connection")
		}
	})
}

func TestQUICDialer(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	t.Run("works as intended", func(t *testing.T) {
		tlsConfig := &tls.Config{
			ServerName: "dns.google",
		}
		d := netxlite.NewQUICDialerWithoutResolver(
			netxlite.NewQUICListener(), log.Log,
		)
		ctx := context.Background()
		sess, err := d.DialContext(
			ctx, "udp", "8.8.8.8:443", tlsConfig, &quic.Config{})
		if err != nil {
			t.Fatal("not the error we expected", err)
		}
		<-sess.HandshakeComplete().Done()
		if err := sess.CloseWithError(0, ""); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("can guess the SNI and ALPN when using a domain name for web", func(t *testing.T) {
		d := netxlite.NewQUICDialerWithResolver(
			netxlite.NewQUICListener(), log.Log,
			netxlite.NewResolverStdlib(log.Log),
		)
		ctx := context.Background()
		sess, err := d.DialContext(
			ctx, "udp", "dns.google:443", &tls.Config{}, &quic.Config{})
		if err != nil {
			t.Fatal("not the error we expected", err)
		}
		<-sess.HandshakeComplete().Done()
		if err := sess.CloseWithError(0, ""); err != nil {
			t.Fatal(err)
		}
	})
}
