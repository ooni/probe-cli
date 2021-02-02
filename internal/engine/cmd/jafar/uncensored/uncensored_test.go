package uncensored

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
)

func TestGood(t *testing.T) {
	client, err := NewClient("dot://1.1.1.1:853")
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()
	if client.Address() != "1.1.1.1:853" {
		t.Fatal("invalid address")
	}
	if client.Network() != "dot" {
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
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("Google is built by a large team")) {
		t.Fatal("not the expected body")
	}
}

func TestNewClientFailure(t *testing.T) {
	clnt, err := NewClient("antani:///")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if clnt != nil {
		t.Fatal("expected nil client here")
	}
}
