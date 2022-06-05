package tracex

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
)

func TestMaybeWrapHTTPTransport(t *testing.T) {
	const snapshotSize = 1024

	t.Run("with non-nil saver", func(t *testing.T) {
		saver := &Saver{}
		underlying := &mocks.HTTPTransport{}
		txp := saver.MaybeWrapHTTPTransport(underlying, snapshotSize)
		realTxp := txp.(*HTTPTransportSaver)
		if realTxp.HTTPTransport != underlying {
			t.Fatal("unexpected result")
		}
		if realTxp.SnapshotSize != snapshotSize {
			t.Fatal("did not set snapshotSize correctly")
		}
	})

	t.Run("with nil saver", func(t *testing.T) {
		var saver *Saver
		underlying := &mocks.HTTPTransport{}
		txp := saver.MaybeWrapHTTPTransport(underlying, snapshotSize)
		if txp != underlying {
			t.Fatal("unexpected result")
		}
	})
}

func TestHTTPTransportSaver(t *testing.T) {

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.HTTPTransport{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		dialer := &HTTPTransportSaver{
			HTTPTransport: child,
			Saver:         &Saver{},
		}
		dialer.CloseIdleConnections()
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
		dialer := &HTTPTransportSaver{
			HTTPTransport: child,
			Saver:         &Saver{},
		}
		if dialer.Network() != expected {
			t.Fatal("unexpected Network")
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {

		measureHTTP := func(t *testing.T, URL *url.URL) (*http.Response, *Saver, error) {
			saver := &Saver{}
			txp := &HTTPTransportSaver{
				HTTPTransport: netxlite.NewHTTPTransportStdlib(model.DiscardLogger),
				Saver:         saver,
			}
			req, err := http.NewRequest("GET", URL.String(), nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Add("User-Agent", "miniooni")
			resp, err := txp.RoundTrip(req)
			return resp, saver, err
		}

		validateRequestFields := func(t *testing.T, value *EventValue, URL *url.URL) {
			if value.HTTPMethod != "GET" {
				t.Fatal("invalid method")
			}
			if value.HTTPRequestHeaders.Get("Host") != URL.Host {
				t.Fatal("invalid Host header")
			}
			if value.HTTPRequestHeaders.Get("User-Agent") != "miniooni" {
				t.Fatal("invalid User-Agent header")
			}
			if value.HTTPURL != URL.String() {
				t.Fatal("invalid URL")
			}
			if value.Time.IsZero() {
				t.Fatal("expected nonzero Time")
			}
			if value.Transport != "tcp" {
				t.Fatal("expected Transport to be tcp")
			}
		}

		validateRequest := func(t *testing.T, ev Event, URL *url.URL) {
			if _, good := ev.(*EventHTTPTransactionStart); !good {
				t.Fatal("invalid event type")
			}
			if ev.Name() != "http_transaction_start" {
				t.Fatal("invalid event name")
			}
			value := ev.Value()
			validateRequestFields(t, value, URL)
		}

		validateResponseSuccess := func(t *testing.T, ev Event, URL *url.URL) {
			if _, good := ev.(*EventHTTPTransactionDone); !good {
				t.Fatal("invalid event type")
			}
			if ev.Name() != "http_transaction_done" {
				t.Fatal("invalid event name")
			}
			value := ev.Value()
			validateRequestFields(t, value, URL)
			if value.Duration <= 0 {
				t.Fatal("expected nonzero duration")
			}
			if len(value.HTTPResponseHeaders) <= 0 {
				t.Fatal("expected at least one response header")
			}
			if !bytes.Equal(value.HTTPResponseBody, filtering.HTTPBlockpage451) {
				t.Fatal("unexpected value for response body")
			}
			if value.HTTPStatusCode != 451 {
				t.Fatal("unexpected status code")
			}
		}

		t.Run("on success", func(t *testing.T) {
			server := filtering.NewHTTPServerCleartext(filtering.HTTPAction451)
			defer server.Close()
			resp, saver, err := measureHTTP(t, server.URL())
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 451 {
				t.Fatal("unexpected status code", resp.StatusCode)
			}
			events := saver.Read()
			if len(events) != 2 {
				t.Fatal("unexpected number of events")
			}
			validateRequest(t, events[0], server.URL())
			validateResponseSuccess(t, events[1], server.URL())
			data, err := netxlite.ReadAllContext(context.Background(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data, filtering.HTTPBlockpage451) {
				t.Fatal("we cannot re-read the same body")
			}
		})

		validateResponseFailure := func(t *testing.T, ev Event, URL *url.URL) {
			if _, good := ev.(*EventHTTPTransactionDone); !good {
				t.Fatal("invalid event type")
			}
			if ev.Name() != "http_transaction_done" {
				t.Fatal("invalid event name")
			}
			value := ev.Value()
			validateRequestFields(t, value, URL)
			if value.Duration <= 0 {
				t.Fatal("expected nonzero duration")
			}
			if value.Err != netxlite.FailureConnectionReset {
				t.Fatal("unexpected Err value")
			}
			if len(value.HTTPResponseHeaders) > 0 {
				t.Fatal("expected zero response headers")
			}
			if !bytes.Equal(value.HTTPResponseBody, nil) {
				t.Fatal("unexpected value for response body")
			}
			if value.HTTPStatusCode != 0 {
				t.Fatal("unexpected status code")
			}
		}

		t.Run("on round trip failure", func(t *testing.T) {
			server := filtering.NewHTTPServerCleartext(filtering.HTTPActionReset)
			defer server.Close()
			resp, saver, err := measureHTTP(t, server.URL())
			if err == nil || err.Error() != "connection_reset" {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("expected nil response")
			}
			events := saver.Read()
			if len(events) != 2 {
				t.Fatal("unexpected number of events")
			}
			validateRequest(t, events[0], server.URL())
			validateResponseFailure(t, events[1], server.URL())
		})

		// Sometimes useful for testing
		/*
			dump := func(t *testing.T, ev Event) {
				data, _ := json.MarshalIndent(ev.Value(), " ", " ")
				t.Log(string(data))
				t.Fail()
			}
		*/

		t.Run("on error reading the response body", func(t *testing.T) {
			saver := &Saver{}
			expected := errors.New("mocked error")
			txp := HTTPTransportSaver{
				HTTPTransport: &mocks.HTTPTransport{
					MockRoundTrip: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							Header: http.Header{
								"Server": {"antani"},
							},
							StatusCode: 200,
							Body: io.NopCloser(&mocks.Reader{
								MockRead: func(b []byte) (int, error) {
									return 0, expected
								},
							}),
						}, nil
					},
					MockNetwork: func() string {
						return "tcp"
					},
				},
				SnapshotSize: 4,
				Saver:        saver,
			}
			URL := &url.URL{
				Scheme: "http",
				Host:   "127.0.0.1:9050",
			}
			req, err := http.NewRequest("GET", URL.String(), nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Add("User-Agent", "miniooni")
			resp, err := txp.RoundTrip(req)
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected")
			}
			if resp != nil {
				t.Fatal("expected nil response")
			}
			ev := saver.Read()
			validateRequest(t, ev[0], URL)
			if ev[1].Value().HTTPStatusCode != 200 {
				t.Fatal("invalid status code")
			}
			if ev[1].Value().HTTPResponseHeaders.Get("Server") != "antani" {
				t.Fatal("invalid Server header")
			}
			if ev[1].Value().Err != "unknown_failure: mocked error" {
				t.Fatal("invalid error")
			}
		})
	})
}

func TestHTTPCloneRequestHeaders(t *testing.T) {
	t.Run("with req.Host set", func(t *testing.T) {
		req := &http.Request{
			Host: "www.example.com",
			URL: &url.URL{
				Host: "www.kernel.org",
			},
			Header: http.Header{},
		}
		header := httpCloneRequestHeaders(req)
		if header.Get("Host") != "www.example.com" {
			t.Fatal("did not set Host header correctly")
		}
	})

	t.Run("with only req.URL.Host set", func(t *testing.T) {
		req := &http.Request{
			Host: "",
			URL: &url.URL{
				Host: "www.kernel.org",
			},
			Header: http.Header{},
		}
		header := httpCloneRequestHeaders(req)
		if header.Get("Host") != "www.kernel.org" {
			t.Fatal("did not set Host header correctly")
		}
	})
}
