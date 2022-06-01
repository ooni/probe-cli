package tracex

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
)

func TestSaverTransactionHTTPTransport(t *testing.T) {

	startServer := func(t *testing.T, action filtering.HTTPAction) (net.Listener, *url.URL) {
		server := &filtering.HTTPProxy{
			OnIncomingHost: func(host string) filtering.HTTPAction {
				return action
			},
		}
		listener, err := server.Start("127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		URL := &url.URL{
			Scheme: "http",
			Host:   listener.Addr().String(),
			Path:   "/",
		}
		return listener, URL
	}

	measureHTTP := func(t *testing.T, URL *url.URL) (*http.Response, *Saver, error) {
		saver := &Saver{}
		txp := &SaverTransactionHTTPTransport{
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
		listener, URL := startServer(t, filtering.HTTPAction451)
		defer listener.Close()
		resp, saver, err := measureHTTP(t, URL)
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
		validateRequest(t, events[0], URL)
		validateResponseSuccess(t, events[1], URL)
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
		if value.Err.Error() != "connection_reset" {
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
		listener, URL := startServer(t, filtering.HTTPActionReset)
		defer listener.Close()
		resp, saver, err := measureHTTP(t, URL)
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
		validateRequest(t, events[0], URL)
		validateResponseFailure(t, events[1], URL)
	})

	// Sometimes useful for testing
	/*
		dumplog := func(t *testing.T, ev Event) {
			data, _ := json.MarshalIndent(ev.Value(), " ", " ")
			t.Log(string(data))
			t.FailNow()
		}
	*/

	t.Run("on error reading the response body", func(t *testing.T) {
		saver := &Saver{}
		expected := errors.New("mocked error")
		txp := SaverTransactionHTTPTransport{
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
		if ev[1].Value().Err.Error() != "unknown_failure: mocked error" {
			t.Fatal("invalid error")
		}
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
