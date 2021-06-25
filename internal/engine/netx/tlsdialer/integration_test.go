package tlsdialer_test

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestTLSDialerSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	log.SetLevel(log.DebugLevel)
	dialer := tlsdialer.TLSDialer{Dialer: new(net.Dialer),
		TLSHandshaker: tlsdialer.LoggingTLSHandshaker{
			TLSHandshaker: &netxlite.TLSHandshakerStdlib{},
			Logger:        log.Log,
		},
	}
	txp := &http.Transport{DialTLS: func(network, address string) (net.Conn, error) {
		// AlpineLinux edge is still using Go 1.13. We cannot switch to
		// using DialTLSContext here as we'd like to until either Alpine
		// switches to Go 1.14 or we drop the MK dependency.
		return dialer.DialTLSContext(context.Background(), network, address)
	}}
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
