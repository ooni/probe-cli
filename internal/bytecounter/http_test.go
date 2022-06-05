package bytecounter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestMaybeWrapHTTPTransport(t *testing.T) {
	t.Run("when counter is not nil", func(t *testing.T) {
		underlying := &mocks.HTTPTransport{}
		counter := &Counter{}
		txp := counter.MaybeWrapHTTPTransport(underlying)
		realTxp := txp.(*httpTransport)
		if realTxp.HTTPTransport != underlying {
			t.Fatal("did not wrap correctly")
		}
	})

	t.Run("when counter is nil", func(t *testing.T) {
		underlying := &mocks.HTTPTransport{}
		var counter *Counter
		txp := counter.MaybeWrapHTTPTransport(underlying)
		if txp != underlying {
			t.Fatal("unexpected result")
		}
	})
}

func TestHTTPTransport(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("failure", func(t *testing.T) {
			counter := New()
			txp := &httpTransport{
				Counter: counter,
				HTTPTransport: &mocks.HTTPTransport{
					MockRoundTrip: func(req *http.Request) (*http.Response, error) {
						return nil, io.EOF
					},
				},
			}
			req, err := http.NewRequest(
				"POST", "https://www.google.com", strings.NewReader("AAAAAA"))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("User-Agent", "antani-browser/1.0.0")
			resp, err := txp.RoundTrip(req)
			if !errors.Is(err, io.EOF) {
				t.Fatal("not the error we expected")
			}
			if resp != nil {
				t.Fatal("expected nil response here")
			}
			if counter.Sent.Load() != 62 {
				t.Fatal("expected 62 bytes sent", counter.Sent.Load())
			}
			if counter.Received.Load() != 0 {
				t.Fatal("expected zero bytes received", counter.Received.Load())
			}
		})

		t.Run("success", func(t *testing.T) {
			counter := New()
			txp := &httpTransport{
				Counter: counter,
				HTTPTransport: &mocks.HTTPTransport{
					MockRoundTrip: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body: io.NopCloser(strings.NewReader("1234567")),
							Header: http.Header{
								"Server": []string{"antani/0.1.0"},
							},
							Status:     "200 OK",
							StatusCode: http.StatusOK,
						}
						return resp, nil
					},
				},
			}
			req, err := http.NewRequest(
				"POST", "https://www.google.com", strings.NewReader("AAAAAA"))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("User-Agent", "antani-browser/1.0.0")
			resp, err := txp.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			data, err := netxlite.ReadAllContext(context.Background(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if string(data) != "1234567" {
				t.Fatal("expected a different body here")
			}
			if counter.Sent.Load() != 62 {
				t.Fatal("expected 62 bytes sent", counter.Sent.Load())
			}
			if counter.Received.Load() != 37 {
				t.Fatal("expected 37 bytes received", counter.Received.Load())
			}
		})

		t.Run("success with EOF", func(t *testing.T) {
			counter := New()
			txp := &httpTransport{
				Counter: counter,
				HTTPTransport: &mocks.HTTPTransport{
					MockRoundTrip: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body: io.NopCloser(&mocks.Reader{
								MockRead: func(b []byte) (int, error) {
									if len(b) < 1 {
										panic("should not happen")
									}
									b[0] = 'A'
									return 1, io.EOF // we want code to be robust to this
								},
							}),
							Header: http.Header{
								"Server": []string{"antani/0.1.0"},
							},
							Status:     "200 OK",
							StatusCode: http.StatusOK,
						}
						return resp, nil
					},
				},
			}
			client := &http.Client{Transport: txp}
			resp, err := client.Get("https://www.google.com")
			if err != nil {
				t.Fatal(err)
			}
			data, err := netxlite.ReadAllContext(context.Background(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if string(data) != "A" {
				t.Fatal("expected a different body here")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.HTTPTransport{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		counter := New()
		txp := WrapHTTPTransport(child, counter)
		txp.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("Network", func(t *testing.T) {
		expected := "antani"
		child := &mocks.HTTPTransport{
			MockNetwork: func() string {
				return expected
			},
		}
		counter := New()
		txp := WrapHTTPTransport(child, counter)
		if network := txp.Network(); network != expected {
			t.Fatal("unexpected network", network)
		}
	})
}
