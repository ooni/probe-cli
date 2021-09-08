package netxlite_test

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
)

func TestHTTPTransport(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	t.Run("works as intended", func(t *testing.T) {
		d := netxlite.NewDialerWithResolver(log.Log, netxlite.NewResolverSystem(log.Log))
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
			netxlite.NewResolverSystem(log.Log),
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
