package netxlite

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestDNSOverHTTPSTransport(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("query serialization failure", func(t *testing.T) {
			txp := NewDNSOverHTTPSTransport(http.DefaultClient, "https://1.1.1.1/dns-query")
			expected := errors.New("mocked error")
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return nil, expected
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("expected no response here")
			}
		})

		t.Run("NewRequestFailure", func(t *testing.T) {
			const invalidURL = "\t"
			txp := NewDNSOverHTTPSTransport(http.DefaultClient, invalidURL)
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 17), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
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
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 17), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
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
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 17), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if err == nil || err.Error() != "doh: server returned error" {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
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
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 17), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if err == nil || err.Error() != "doh: invalid content-type" {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("expected no response here")
			}
		})

		t.Run("ReadAllContext fails", func(t *testing.T) {
			expected := errors.New("mocked error")
			txp := &DNSOverHTTPSTransport{
				Client: &mocks.HTTPClient{
					MockDo: func(*http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body: io.NopCloser(&mocks.Reader{
								MockRead: func(b []byte) (int, error) {
									return 0, expected
								},
							}),
							Header: http.Header{
								"Content-Type": []string{"application/dns-message"},
							},
						}, nil
					},
				},
				URL: "https://cloudflare-dns.com/dns-query",
			}
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 17), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("expected no response here")
			}
		})

		t.Run("decode response failure", func(t *testing.T) {
			expected := errors.New("mocked error")
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
				Decoder: &mocks.DNSDecoder{
					MockDecodeResponse: func(data []byte, query model.DNSQuery) (model.DNSResponse, error) {
						return nil, expected
					},
				},
			}
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 17), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
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
				Decoder: &mocks.DNSDecoder{
					MockDecodeResponse: func(data []byte, query model.DNSQuery) (model.DNSResponse, error) {
						return &mocks.DNSResponse{}, nil
					},
				},
			}
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 17), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if err != nil {
				t.Fatal(err)
			}
			if resp == nil {
				t.Fatal("expected non-nil resp here")
			}
		})

		t.Run("sets the correct user-agent", func(t *testing.T) {
			expected := errors.New("mocked error")
			var correct bool
			txp := &DNSOverHTTPSTransport{
				Client: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						correct = req.Header.Get("User-Agent") == model.HTTPHeaderUserAgent
						return nil, expected
					},
				},
				URL: "https://cloudflare-dns.com/dns-query",
			}
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 17), nil
				},
			}
			data, err := txp.RoundTrip(context.Background(), query)
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
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 17), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, expected) {
				t.Fatal("expected an error here")
			}
			if resp != nil {
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
