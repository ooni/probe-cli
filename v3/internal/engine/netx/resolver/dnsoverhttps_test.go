package resolver_test

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/httpheader"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
)

func TestDNSOverHTTPSNewRequestFailure(t *testing.T) {
	const invalidURL = "\t"
	txp := resolver.NewDNSOverHTTPS(http.DefaultClient, invalidURL)
	data, err := txp.RoundTrip(context.Background(), nil)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("expected an error here")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverHTTPSClientDoFailure(t *testing.T) {
	expected := errors.New("mocked error")
	txp := resolver.DNSOverHTTPS{
		Do: func(*http.Request) (*http.Response, error) {
			return nil, expected
		},
		URL: "https://cloudflare-dns.com/dns-query",
	}
	data, err := txp.RoundTrip(context.Background(), nil)
	if !errors.Is(err, expected) {
		t.Fatal("expected an error here")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverHTTPSHTTPFailure(t *testing.T) {
	txp := resolver.DNSOverHTTPS{
		Do: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 500,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil
		},
		URL: "https://cloudflare-dns.com/dns-query",
	}
	data, err := txp.RoundTrip(context.Background(), nil)
	if err == nil || err.Error() != "doh: server returned error" {
		t.Fatal("expected an error here")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverHTTPSMissingContentType(t *testing.T) {
	txp := resolver.DNSOverHTTPS{
		Do: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil
		},
		URL: "https://cloudflare-dns.com/dns-query",
	}
	data, err := txp.RoundTrip(context.Background(), nil)
	if err == nil || err.Error() != "doh: invalid content-type" {
		t.Fatal("expected an error here")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverHTTPSSuccess(t *testing.T) {
	body := []byte("AAA")
	txp := resolver.DNSOverHTTPS{
		Do: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewReader(body)),
				Header: http.Header{
					"Content-Type": []string{"application/dns-message"},
				},
			}, nil
		},
		URL: "https://cloudflare-dns.com/dns-query",
	}
	data, err := txp.RoundTrip(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, body) {
		t.Fatal("not the response we expected")
	}
}

func TestDNSOverHTTPTransportOK(t *testing.T) {
	const queryURL = "https://cloudflare-dns.com/dns-query"
	txp := resolver.NewDNSOverHTTPS(http.DefaultClient, queryURL)
	if txp.Network() != "doh" {
		t.Fatal("invalid network")
	}
	if txp.RequiresPadding() != true {
		t.Fatal("should require padding")
	}
	if txp.Address() != queryURL {
		t.Fatal("invalid address")
	}
}

func TestDNSOverHTTPSClientSetsUserAgent(t *testing.T) {
	expected := errors.New("mocked error")
	var correct bool
	txp := resolver.DNSOverHTTPS{
		Do: func(req *http.Request) (*http.Response, error) {
			correct = req.Header.Get("User-Agent") == httpheader.UserAgent()
			return nil, expected
		},
		URL: "https://cloudflare-dns.com/dns-query",
	}
	data, err := txp.RoundTrip(context.Background(), nil)
	if !errors.Is(err, expected) {
		t.Fatal("expected an error here")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
	if !correct {
		t.Fatal("did not see correct user agent")
	}
}

func TestDNSOverHTTPSHostOverride(t *testing.T) {
	var correct bool
	expected := errors.New("mocked error")

	hostOverride := "test.com"
	txp := resolver.DNSOverHTTPS{
		Do: func(req *http.Request) (*http.Response, error) {
			correct = req.Host == hostOverride
			return nil, expected
		},
		URL:          "https://cloudflare-dns.com/dns-query",
		HostOverride: hostOverride,
	}
	data, err := txp.RoundTrip(context.Background(), nil)
	if !errors.Is(err, expected) {
		t.Fatal("expected an error here")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
	if !correct {
		t.Fatal("did not see correct host override")
	}
}
