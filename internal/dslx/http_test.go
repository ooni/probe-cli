package dslx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestHTTPNewRequest(t *testing.T) {
	t.Run("without any option and with domain", func(t *testing.T) {
		ctx := context.Background()
		conn := &HTTPConnection{
			Address:               "130.192.91.211:443",
			Domain:                "example.com",
			Network:               "tcp",
			Scheme:                "https",
			TLSNegotiatedProtocol: "h2",
			Trace:                 nil,
			Transport:             nil,
		}

		req, err := httpNewRequest(ctx, conn, model.DiscardLogger)
		if err != nil {
			t.Fatal(err)
		}

		if req.URL.Scheme != "https" {
			t.Fatal("unexpected req.URL.Scheme", req.URL.Scheme)
		}
		if req.URL.Host != "example.com" {
			t.Fatal("unexpected req.URL.Host", req.URL.Host)
		}
		if req.URL.Path != "/" {
			t.Fatal("unexpected req.URL.Path", req.URL.Path)
		}
		if req.Method != "GET" {
			t.Fatal("unexpected req.Method", req.Method)
		}
		if req.Host != "example.com" {
			t.Fatal("unexpected req.Host", req.Host)
		}
		headers := http.Header{
			"Host": {"example.com"},
		}
		if diff := cmp.Diff(headers, req.Header); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("without any option, without domain but with standard port", func(t *testing.T) {
		ctx := context.Background()
		conn := &HTTPConnection{
			Address:               "130.192.91.211:443",
			Domain:                "",
			Network:               "tcp",
			Scheme:                "https",
			TLSNegotiatedProtocol: "h2",
			Trace:                 nil,
			Transport:             nil,
		}

		req, err := httpNewRequest(ctx, conn, model.DiscardLogger)
		if err != nil {
			t.Fatal(err)
		}

		if req.URL.Scheme != "https" {
			t.Fatal("unexpected req.URL.Scheme", req.URL.Scheme)
		}
		if req.URL.Host != "130.192.91.211" {
			t.Fatal("unexpected req.URL.Host", req.URL.Host)
		}
		if req.URL.Path != "/" {
			t.Fatal("unexpected req.URL.Path", req.URL.Path)
		}
		if req.Method != "GET" {
			t.Fatal("unexpected req.Method", req.Method)
		}
		if req.Host != "130.192.91.211" {
			t.Fatal("unexpected req.Host", req.Host)
		}
		headers := http.Header{
			"Host": {"130.192.91.211"},
		}
		if diff := cmp.Diff(headers, req.Header); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("without any option, without domain but with nonstandard port", func(t *testing.T) {
		ctx := context.Background()
		conn := &HTTPConnection{
			Address:               "130.192.91.211:443",
			Domain:                "",
			Network:               "tcp",
			Scheme:                "http",
			TLSNegotiatedProtocol: "h2",
			Trace:                 nil,
			Transport:             nil,
		}

		req, err := httpNewRequest(ctx, conn, model.DiscardLogger)
		if err != nil {
			t.Fatal(err)
		}

		if req.URL.Scheme != "http" {
			t.Fatal("unexpected req.URL.Scheme", req.URL.Scheme)
		}
		if req.URL.Host != "130.192.91.211:443" {
			t.Fatal("unexpected req.URL.Host", req.URL.Host)
		}
		if req.URL.Path != "/" {
			t.Fatal("unexpected req.URL.Path", req.URL.Path)
		}
		if req.Method != "GET" {
			t.Fatal("unexpected req.Method", req.Method)
		}
		if req.Host != "130.192.91.211:443" {
			t.Fatal("unexpected req.Host", req.Host)
		}
		headers := http.Header{
			"Host": {"130.192.91.211:443"},
		}
		if diff := cmp.Diff(headers, req.Header); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("with all options", func(t *testing.T) {
		ctx := context.Background()
		conn := &HTTPConnection{
			Address:               "130.192.91.211:443",
			Domain:                "example.com",
			Network:               "tcp",
			Scheme:                "https",
			TLSNegotiatedProtocol: "h2",
			Trace:                 nil,
			Transport:             nil,
		}

		options := []HTTPRequestOption{
			HTTPRequestOptionAccept("text/html"),
			HTTPRequestOptionAcceptLanguage("de"),
			HTTPRequestOptionHost("www.x.org"),
			HTTPRequestOptionMethod("PUT"),
			HTTPRequestOptionReferer("https://example.com/"),
			HTTPRequestOptionURLPath("/path/to/example"),
			HTTPRequestOptionUserAgent("Mozilla/5.0 Gecko/geckotrail Firefox/firefoxversion"),
		}

		req, err := httpNewRequest(ctx, conn, model.DiscardLogger, options...)
		if err != nil {
			t.Fatal(err)
		}

		if req.URL.Scheme != "https" {
			t.Fatal("unexpected req.URL.Scheme", req.URL.Scheme)
		}
		if req.URL.Host != "www.x.org" {
			t.Fatal("unexpected req.URL.Host", req.URL.Host)
		}
		if req.URL.Path != "/path/to/example" {
			t.Fatal("unexpected req.URL.Path", req.URL.Path)
		}
		if req.Method != "PUT" {
			t.Fatal("unexpected req.Method", req.Method)
		}
		if req.Host != "www.x.org" {
			t.Fatal("unexpected req.Host", req.Host)
		}
		headers := http.Header{
			"Accept":          {"text/html"},
			"Accept-Language": {"de"},
			"Host":            {"www.x.org"},
			"Referer":         {"https://example.com/"},
			"User-Agent":      {"Mozilla/5.0 Gecko/geckotrail Firefox/firefoxversion"},
		}
		if diff := cmp.Diff(headers, req.Header); diff != "" {
			t.Fatal(diff)
		}
	})
}

/*
Test cases:
- Apply httpRequestFunc:
  - with EOF
  - with invalid method
  - with port-less address
  - with success (https)
  - with success (http)
  - with header options
*/
func TestHTTPRequest(t *testing.T) {
	t.Run("Apply httpRequestFunc", func(t *testing.T) {
		mockResponse := &http.Response{
			Status: "expected",
			Body:   io.NopCloser(strings.NewReader("")),
		}

		eofTransport := &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				return nil, io.EOF
			},
			MockNetwork: func() string {
				return "tcp"
			},
		}

		goodTransport := &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				return mockResponse, nil
			},
			MockNetwork: func() string {
				return "tcp"
			},
		}
		idGen := &atomic.Int64{}
		zeroTime := time.Time{}
		trace := measurexlite.NewTrace(idGen.Add(1), zeroTime, "antani")

		t.Run("with EOF", func(t *testing.T) {
			httpTransport := HTTPConnection{
				Address:   "1.2.3.4:567",
				Network:   "tcp",
				Scheme:    "https",
				Trace:     trace,
				Transport: eofTransport,
			}
			httpRequest := HTTPRequest(
				NewMinimalRuntime(model.DiscardLogger, time.Now()),
			)
			res := httpRequest.Apply(context.Background(), NewMaybeWithValue(&httpTransport))
			if res.Error != io.EOF {
				t.Fatal("not the error we expected")
			}
			if res.State.HTTPResponse != nil {
				t.Fatal("expected nil request here")
			}
		})

		t.Run("with invalid domain", func(t *testing.T) {
			httpTransport := HTTPConnection{
				Address:   "1.2.3.4:567",
				Domain:    "\t", // invalid domain
				Network:   "tcp",
				Scheme:    "https",
				Trace:     trace,
				Transport: goodTransport,
			}
			rt := NewMinimalRuntime(model.DiscardLogger, time.Now())
			httpRequest := HTTPRequest(rt)
			res := httpRequest.Apply(context.Background(), NewMaybeWithValue(&httpTransport))
			if res.Error == nil || !strings.HasPrefix(res.Error.Error(), `parse "https://%09/": invalid URL escape "%09"`) {
				t.Fatal("not the error we expected", res.Error)
			}
			if res.State.HTTPResponse != nil {
				t.Fatal("expected nil request here")
			}
		})

		t.Run("with port-less address", func(t *testing.T) {
			httpTransport := HTTPConnection{
				Address:   "1.2.3.4",
				Network:   "tcp",
				Scheme:    "https",
				Trace:     trace,
				Transport: goodTransport,
			}
			httpRequest := HTTPRequest(
				NewMinimalRuntime(model.DiscardLogger, time.Now()),
			)
			res := httpRequest.Apply(context.Background(), NewMaybeWithValue(&httpTransport))
			if res.Error != nil {
				t.Fatal("expected error")
			}
			if res.State.HTTPResponse == nil {
				t.Fatal("unexpected nil request")
			}
			if res.State.HTTPRequest.Host != "1.2.3.4" {
				t.Fatal("unexpected host")
			}
		})

		// makeSureObservationsContainTags ensures the observations you can extract from
		// the given HTTPResponse contain the tags we configured when testing
		makeSureObservationsContainTags := func(res *Maybe[*HTTPResponse]) error {
			// exclude the case where there was an error
			if res.Error != nil {
				return fmt.Errorf("unexpected error: %w", res.Error)
			}

			// obtain the observations
			for _, obs := range ExtractObservations(res) {

				// check the network events
				for _, ev := range obs.NetworkEvents {
					if diff := cmp.Diff([]string{"antani"}, ev.Tags); diff != "" {
						return errors.New(diff)
					}
				}

				// check the HTTP events
				for _, ev := range obs.Requests {
					if diff := cmp.Diff([]string{"antani"}, ev.Tags); diff != "" {
						return errors.New(diff)
					}
				}
			}

			return nil
		}

		t.Run("with success (https)", func(t *testing.T) {
			httpTransport := HTTPConnection{
				Address:   "1.2.3.4:443",
				Network:   "tcp",
				Scheme:    "https",
				Trace:     trace,
				Transport: goodTransport,
			}
			httpRequest := HTTPRequest(
				NewMinimalRuntime(model.DiscardLogger, time.Now()),
			)
			res := httpRequest.Apply(context.Background(), NewMaybeWithValue(&httpTransport))
			if res.Error != nil {
				t.Fatal("unexpected error")
			}
			if res.State.HTTPResponse == nil || res.State.HTTPResponse.Status != "expected" {
				t.Fatal("unexpected request")
			}
			makeSureObservationsContainTags(res)
		})

		t.Run("with success (http)", func(t *testing.T) {
			httpTransport := HTTPConnection{
				Address:   "1.2.3.4:80",
				Network:   "tcp",
				Scheme:    "http",
				Trace:     trace,
				Transport: goodTransport,
			}
			httpRequest := HTTPRequest(
				NewMinimalRuntime(model.DiscardLogger, time.Now()),
			)
			res := httpRequest.Apply(context.Background(), NewMaybeWithValue(&httpTransport))
			if res.Error != nil {
				t.Fatal("unexpected error")
			}
			if res.State.HTTPResponse == nil || res.State.HTTPResponse.Status != "expected" {
				t.Fatal("unexpected request")
			}
			makeSureObservationsContainTags(res)
		})

		t.Run("with header options", func(t *testing.T) {
			httpTransport := HTTPConnection{
				Address:   "1.2.3.4:567",
				Domain:    "domain.com",
				Network:   "tcp",
				Scheme:    "https",
				Trace:     trace,
				Transport: goodTransport,
			}
			rt := NewMinimalRuntime(model.DiscardLogger, time.Now())
			httpRequest := HTTPRequest(rt,
				HTTPRequestOptionAccept("text/html"),
				HTTPRequestOptionAcceptLanguage("de"),
				HTTPRequestOptionHost("host"),
				HTTPRequestOptionReferer("https://example.org"),
				HTTPRequestOptionURLPath("/path/to/example"),
				HTTPRequestOptionUserAgent("Mozilla/5.0 Gecko/geckotrail Firefox/firefoxversion"),
			)
			res := httpRequest.Apply(context.Background(), NewMaybeWithValue(&httpTransport))
			if res.Error != nil {
				t.Fatal("unexpected error")
			}
			if res.State.HTTPResponse == nil || res.State.HTTPResponse.Status != "expected" {
				t.Fatal("unexpected request")
			}
			if res.State.HTTPRequest.Header.Get("Accept") != "text/html" ||
				res.State.HTTPRequest.Header.Get("Accept-Language") != "de" ||
				res.State.HTTPRequest.Header.Get("Host") != "host" ||
				res.State.HTTPRequest.Header.Get("Referer") != "https://example.org" ||
				res.State.HTTPRequest.Header.Get("User-Agent") != "Mozilla/5.0 Gecko/geckotrail Firefox/firefoxversion" {
				t.Fatal("unexpected request header")
			}
			if res.State.HTTPRequest.URL.Path != "/path/to/example" {
				t.Fatal("unexpected URL path", res.State.HTTPRequest.URL.Path)
			}
		})
	})
}

/*
Test cases:
- Get composed function: TCP with HTTP
- Apply httpTransportTCPFunc
*/
func TestHTTPTCP(t *testing.T) {
	t.Run("Get composed function: TCP with HTTP", func(t *testing.T) {
		rt := NewMinimalRuntime(model.DiscardLogger, time.Now())
		f := HTTPRequestOverTCP(rt)
		if _, ok := f.(*compose2Func[*TCPConnection, *HTTPConnection, *HTTPResponse]); !ok {
			t.Fatal("unexpected type")
		}
	})

	t.Run("Apply httpTransportTCPFunc", func(t *testing.T) {
		conn := &mocks.Conn{}
		idGen := &atomic.Int64{}
		zeroTime := time.Time{}
		trace := measurexlite.NewTrace(idGen.Add(1), zeroTime)
		address := "1.2.3.4:567"
		tcpConn := &TCPConnection{
			Address: address,
			Conn:    conn,
			Network: "tcp",
			Trace:   trace,
		}
		f := HTTPConnectionTCP(
			NewMinimalRuntime(model.DiscardLogger, time.Now()),
		)
		res := f.Apply(context.Background(), NewMaybeWithValue(tcpConn))
		if res.Error != nil {
			t.Fatalf("unexpected error: %s", res.Error)
		}
		if res.State == nil {
			t.Fatal("unexpected nil transport")
		}
		if res.State.Scheme != "http" {
			t.Fatalf("unexpected scheme, want %s, got %s", "http", res.State.Scheme)
		}
		if res.State.Address != address {
			t.Fatalf("unexpected address, want %s, got %s", address, res.State.Address)
		}
	})
}

/*
Test cases:
- Get composed function: QUIC with HTTP
- Apply httpTransportQUICFunc
*/
func TestHTTPQUIC(t *testing.T) {
	t.Run("Get composed function: QUIC with HTTP", func(t *testing.T) {
		rt := NewMinimalRuntime(model.DiscardLogger, time.Now())
		f := HTTPRequestOverQUIC(rt)
		if _, ok := f.(*compose2Func[*QUICConnection, *HTTPConnection, *HTTPResponse]); !ok {
			t.Fatal("unexpected type")
		}
	})

	t.Run("Apply httpTransportQUICFunc", func(t *testing.T) {
		conn := &mocks.QUICEarlyConnection{}
		idGen := &atomic.Int64{}
		zeroTime := time.Time{}
		trace := measurexlite.NewTrace(idGen.Add(1), zeroTime)
		address := "1.2.3.4:567"
		quicConn := &QUICConnection{
			Address:  address,
			QUICConn: conn,
			Network:  "udp",
			Trace:    trace,
		}
		f := HTTPConnectionQUIC(
			NewMinimalRuntime(model.DiscardLogger, time.Now()),
		)
		res := f.Apply(context.Background(), NewMaybeWithValue(quicConn))
		if res.Error != nil {
			t.Fatalf("unexpected error: %s", res.Error)
		}
		if res.State == nil {
			t.Fatal("unexpected nil transport")
		}
		if res.State.Scheme != "https" {
			t.Fatalf("unexpected scheme, want %s, got %s", "https", res.State.Scheme)
		}
		if res.State.Address != address {
			t.Fatalf("unexpected address, want %s, got %s", address, res.State.Address)
		}
	})
}

/*
Test cases:
- Get composed function: TLS with HTTP
- Apply httpTransportTLSFunc
*/
func TestHTTPTLS(t *testing.T) {
	t.Run("Get composed function: TLS with HTTP", func(t *testing.T) {
		rt := NewMinimalRuntime(model.DiscardLogger, time.Now())
		f := HTTPRequestOverTLS(rt)
		if _, ok := f.(*compose2Func[*TLSConnection, *HTTPConnection, *HTTPResponse]); !ok {
			t.Fatal("unexpected type")
		}
	})

	t.Run("Apply httpTransportTLSFunc", func(t *testing.T) {
		conn := &mocks.TLSConn{}
		idGen := &atomic.Int64{}
		zeroTime := time.Time{}
		trace := measurexlite.NewTrace(idGen.Add(1), zeroTime)
		address := "1.2.3.4:567"
		tlsConn := &TLSConnection{
			Address: address,
			Conn:    conn,
			Network: "tcp",
			Trace:   trace,
		}
		f := HTTPConnectionTLS(
			NewMinimalRuntime(model.DiscardLogger, time.Now()),
		)
		res := f.Apply(context.Background(), NewMaybeWithValue(tlsConn))
		if res.Error != nil {
			t.Fatalf("unexpected error: %s", res.Error)
		}
		if res.State == nil {
			t.Fatal("unexpected nil transport")
		}
		if res.State.Scheme != "https" {
			t.Fatalf("unexpected scheme, want %s, got %s", "https", res.State.Scheme)
		}
		if res.State.Address != address {
			t.Fatalf("unexpected address, want %s, got %s", address, res.State.Address)
		}
	})
}
