package dialer

import (
	"net"
	"net/http"
	"testing"
)

func TestShapingDialerGood(t *testing.T) {
	d := &shapingDialer{Dialer: &net.Dialer{}}
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
