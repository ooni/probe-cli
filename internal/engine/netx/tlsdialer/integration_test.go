package tlsdialer_test

import (
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestTLSDialerSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	log.SetLevel(log.DebugLevel)
	dialer := &netxlite.TLSDialer{Dialer: new(net.Dialer),
		TLSHandshaker: &netxlite.TLSHandshakerLogger{
			TLSHandshaker: &netxlite.TLSHandshakerConfigurable{},
			Logger:        log.Log,
		},
	}
	txp := &http.Transport{
		DialTLSContext:    dialer.DialTLSContext,
		ForceAttemptHTTP2: true,
	}
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
