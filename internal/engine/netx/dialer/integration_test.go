package dialer_test

import (
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
)

func TestDNSDialerSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	log.SetLevel(log.DebugLevel)
	dialer := dialer.DNSDialer{
		Dialer: dialer.LoggingDialer{
			Dialer: new(net.Dialer),
			Logger: log.Log,
		},
		Resolver: new(net.Resolver),
	}
	txp := &http.Transport{DialContext: dialer.DialContext}
	client := &http.Client{Transport: txp}
	resp, err := client.Get("http://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
