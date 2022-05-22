package tlsdialer_test

import (
	"net/http"
	"testing"

	"github.com/apex/log"
	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestTLSDialerSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	log.SetLevel(log.DebugLevel)
	dialer := &netxlite.TLSDialerLegacy{Dialer: netxlite.DefaultDialer,
		TLSHandshaker: &netxlite.TLSHandshakerLogger{
			TLSHandshaker: &netxlite.TLSHandshakerConfigurable{},
			DebugLogger:   log.Log,
		},
	}
	txp := &oohttp.StdlibTransport{
		Transport: &oohttp.Transport{
			DialTLSContext:    dialer.DialTLSContext,
			ForceAttemptHTTP2: true,
		},
	}
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
