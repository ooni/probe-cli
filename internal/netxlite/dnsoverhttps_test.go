package netxlite

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestDNSOverHTTPSTransport(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("NewRequestFailure", func(t *testing.T) {
			const invalidURL = "\t"
			txp := NewDNSOverHTTPSTransport(http.DefaultClient, invalidURL)
			data, err := txp.RoundTrip(context.Background(), nil)
			if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
				t.Fatal("expected an error here")
			}
			if data != nil {
				t.Fatal("expected no response here")
			}
		})

		t.Run("client.Do failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			txp := &DNSOverHTTPSTransport{
				Client: &mocks.HTTPClient{
					MockDo: func(*http.Request) (*http.Response, error) {
						return nil, expected
					},
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
		})

		t.Run("server returns 500", func(t *testing.T) {
			txp := &DNSOverHTTPSTransport{
				Client: &mocks.HTTPClient{
					MockDo: func(*http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 500,
							Body:       io.NopCloser(strings.NewReader("")),
						}, nil
					},
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
		})

		t.Run("missing content type", func(t *testing.T) {
			txp := &DNSOverHTTPSTransport{
				Client: &mocks.HTTPClient{
					MockDo: func(*http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader("")),
						}, nil
					},
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
		})

		t.Run("success", func(t *testing.T) {
			body := []byte("AAA")
			txp := &DNSOverHTTPSTransport{
				Client: &mocks.HTTPClient{
					MockDo: func(*http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(bytes.NewReader(body)),
							Header: http.Header{
								"Content-Type": []string{"application/dns-message"},
							},
						}, nil
					},
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
		})

		t.Run("sets the correct user-agent", func(t *testing.T) {
			expected := errors.New("mocked error")
			var correct bool
			txp := &DNSOverHTTPSTransport{
				Client: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						correct = req.Header.Get("User-Agent") == httpheader.UserAgent()
						return nil, expected
					},
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
		})

		t.Run("we can override the Host header", func(t *testing.T) {
			var correct bool
			expected := errors.New("mocked error")
			hostOverride := "test.com"
			txp := &DNSOverHTTPSTransport{
				Client: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						correct = req.Host == hostOverride
						return nil, expected
					},
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
		})

	})

	t.Run("other functions behave correctly", func(t *testing.T) {
		const queryURL = "https://cloudflare-dns.com/dns-query"
		txp := NewDNSOverHTTPSTransport(http.DefaultClient, queryURL)
		if txp.Network() != "doh" {
			t.Fatal("invalid network")
		}
		if txp.RequiresPadding() != true {
			t.Fatal("should require padding")
		}
		if txp.Address() != queryURL {
			t.Fatal("invalid address")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		doh := &DNSOverHTTPSTransport{
			Client: &mocks.HTTPClient{
				MockCloseIdleConnections: func() {
					called = true
				},
			},
		}
		doh.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})
}
