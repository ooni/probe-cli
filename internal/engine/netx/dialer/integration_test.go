package dialer_test

import (
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
)

func TestDialerNewSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	log.SetLevel(log.DebugLevel)
	d := dialer.New(&dialer.Config{Logger: log.Log}, &net.Resolver{})
	txp := &http.Transport{DialContext: d.DialContext}
	client := &http.Client{Transport: txp}
	resp, err := client.Get("http://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
