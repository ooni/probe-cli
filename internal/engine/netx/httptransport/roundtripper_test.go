package httptransport

import (
	"net/http"
	"net/url"
	"testing"

	"golang.org/x/net/http2"
)

func TestGetTransportHTTPS(t *testing.T) {
	txp := http.DefaultTransport.(*http.Transport).Clone()
	rt := newRoundtripper(txp, Config{}).(*roundTripper)
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "www.google.com",
		},
		Header: http.Header{},
	}
	err := rt.getTransport(req)
	if err != nil {
		t.Fatal("unexpected failure")
	}
	transport := rt.transport
	if transport == nil {
		t.Fatal("unexpected nil transport")
	}
	if _, ok := transport.(*http2.Transport); !ok {
		t.Fatal("unexpected transport type")
	}
}

func TestGetTransportHTTP(t *testing.T) {
	txp := http.DefaultTransport.(*http.Transport).Clone()
	rt := newRoundtripper(txp, Config{}).(*roundTripper)
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   "www.google.com",
		},
		Header: http.Header{},
	}
	err := rt.getTransport(req)
	if err != nil {
		t.Fatal("unexpected failure")
	}
	transport := rt.transport
	if transport == nil {
		t.Fatal("unexpected nil transport")
	}
	if _, ok := transport.(*http.Transport); !ok {
		t.Fatal("unexpected transport type")
	}
}

func TestGetTransportInvalidScheme(t *testing.T) {
	txp := http.DefaultTransport.(*http.Transport).Clone()
	rt := newRoundtripper(txp, Config{}).(*roundTripper)
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "x",
			Host:   "www.google.com",
		},
		Header: http.Header{},
	}
	err := rt.getTransport(req)
	if err == nil {
		t.Fatal("expected an error here")
	}
	transport := rt.transport
	if transport != nil {
		t.Fatal("unexpected transport")
	}
}
