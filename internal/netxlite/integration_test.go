package netxlite_test

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
		txp := netxlite.NewHTTP3Transport(d, &tls.Config{})
		client := &http.Client{Transport: txp}
		resp, err := client.Get("https://www.google.com/robots.txt")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		txp.CloseIdleConnections()
	})
}
