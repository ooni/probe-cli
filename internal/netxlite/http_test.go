package netxlite

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/iox"
	"github.com/ooni/probe-cli/v3/internal/netxmocks"
)

func TestHTTPTransportLoggerFailure(t *testing.T) {
	txp := &HTTPTransportLogger{
		Logger: log.Log,
		HTTPTransport: &netxmocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				return nil, io.EOF
			},
		},
	}
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
}

func TestHTTPTransportLoggerFailureWithNoHostHeader(t *testing.T) {
	foundHost := &atomicx.Int64{}
	txp := &HTTPTransportLogger{
		Logger: log.Log,
		HTTPTransport: &netxmocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Host") == "www.google.com" {
					foundHost.Add(1)
				}
				return nil, io.EOF
			},
		},
	}
	req := &http.Request{
		Header: http.Header{},
		URL: &url.URL{
			Scheme: "https",
			Host:   "www.google.com",
			Path:   "/",
		},
	}
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
	if foundHost.Load() != 1 {
		t.Fatal("host header was not added")
	}
}

func TestHTTPTransportLoggerSuccess(t *testing.T) {
	txp := &HTTPTransportLogger{
		Logger: log.Log,
		HTTPTransport: &netxmocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					Body: io.NopCloser(strings.NewReader("")),
					Header: http.Header{
						"Server": []string{"antani/0.1.0"},
					},
					StatusCode: 200,
				}, nil
			},
		},
	}
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	iox.ReadAllContext(context.Background(), resp.Body)
	resp.Body.Close()
}

func TestHTTPTransportLoggerCloseIdleConnections(t *testing.T) {
	calls := &atomicx.Int64{}
	txp := &HTTPTransportLogger{
		HTTPTransport: &netxmocks.HTTPTransport{
			MockCloseIdleConnections: func() {
				calls.Add(1)
			},
		},
		Logger: log.Log,
	}
	txp.CloseIdleConnections()
	if calls.Load() != 1 {
		t.Fatal("not called")
	}
}
