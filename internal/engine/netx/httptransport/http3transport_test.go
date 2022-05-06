package httptransport_test

import (
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/httptransport"
)

func TestNewHTTP3Transport(t *testing.T) {
	// make sure we can create a working transport using this factory.
	txp := httptransport.NewHTTP3Transport(httptransport.Config{})
	req, err := http.NewRequest("GET", "https://google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
