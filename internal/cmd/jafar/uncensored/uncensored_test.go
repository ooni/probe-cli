package uncensored

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://1.1.1.1/dns-query")
	defer client.CloseIdleConnections()
	if client.Address() != "https://1.1.1.1/dns-query" {
		t.Fatal("invalid address")
	}
	if client.Network() != "doh" {
		t.Fatal("invalid network")
	}
	ctx := context.Background()
	addrs, err := client.LookupHost(ctx, "dns.google")
	if err != nil {
		t.Fatal(err)
	}
	var quad8, two8two4 bool
	for _, addr := range addrs {
		quad8 = quad8 || (addr == "8.8.8.8")
		two8two4 = two8two4 || (addr == "8.8.4.4")
	}
	if quad8 != true && two8two4 != true {
		t.Fatal("invalid response")
	}
	conn, err := client.DialContext(ctx, "tcp", "8.8.8.8:853")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	resp, err := client.RoundTrip(&http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "www.google.com",
			Path:   "/humans.txt",
		},
		Header: http.Header{},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal("invalid status-code")
	}
	data, err := netxlite.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("Google is built by a large team")) {
		t.Fatal("not the expected body")
	}
}
