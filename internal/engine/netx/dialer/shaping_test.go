package dialer

import (
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestShapingDialerGood(t *testing.T) {
	d := &shapingDialer{Dialer: netxlite.DefaultDialer}
	txp := &http.Transport{DialContext: d.DialContext}
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com/")
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected nil response here")
	}
	resp.Body.Close()
}
