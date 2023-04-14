package dslx

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

/*
Test cases:
- Get httpRequestFunc with options
- Apply httpRequestFunc:
  - with EOF
  - with invalid method
  - with port-less address
  - with success (https)
  - with success (http)
  - with header options
*/
func TestHTTPRequest(t *testing.T) {
	t.Run("Get httpRequestFunc with options", func(t *testing.T) {
		f := HTTPRequest(
			HTTPRequestOptionAccept("text/html"),
			HTTPRequestOptionAcceptLanguage("de"),
			HTTPRequestOptionHost("host"),
			HTTPRequestOptionMethod("PUT"),
			HTTPRequestOptionReferer("https://example.com/"),
			HTTPRequestOptionURLPath("/path/to/example"),
			HTTPRequestOptionUserAgent("Mozilla/5.0 Gecko/geckotrail Firefox/firefoxversion"),
		)
		var requestFunc *httpRequestFunc
		var ok bool
		if requestFunc, ok = f.(*httpRequestFunc); !ok {
			t.Fatal("unexpected type. Expected: tlsHandshakeFunc")
		}
		if requestFunc.Accept != "text/html" {
			t.Fatalf("unexpected %s, expected %s, got %s", "Accept", "text/html", requestFunc.Accept)
		}
		if requestFunc.AcceptLanguage != "de" {
			t.Fatalf("unexpected %s, expected %s, got %s", "AcceptLanguage", "de", requestFunc.AcceptLanguage)
		}
		if requestFunc.Host != "host" {
			t.Fatalf("unexpected %s, expected %s, got %s", "Host", "host", requestFunc.Host)
		}
		if requestFunc.Method != "PUT" {
			t.Fatalf("unexpected %s, expected %s, got %s", "Method", "PUT", requestFunc.Method)
		}
		if requestFunc.Referer != "https://example.com/" {
			t.Fatalf("unexpected %s, expected %s, got %s", "Referer", "https://example.com/", requestFunc.Referer)
		}
		if requestFunc.URLPath != "/path/to/example" {
			t.Fatalf("unexpected %s, expected %s, got %s", "URLPath", "example/to/path", requestFunc.URLPath)
		}
		if requestFunc.UserAgent != "Mozilla/5.0 Gecko/geckotrail Firefox/firefoxversion" {
			t.Fatalf("unexpected %s, expected %s, got %s", "UserAgent", "Mozilla/5.0 Gecko/geckotrail Firefox/firefoxversion", requestFunc.UserAgent)
		}
	})

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
		trace := measurexlite.NewTrace(idGen.Add(1), zeroTime)

		t.Run("with EOF", func(t *testing.T) {
			httpTransport := HTTPTransport{
				Address:     "1.2.3.4:567",
				IDGenerator: idGen,
				Logger:      model.DiscardLogger,
				Network:     "tcp",
				Scheme:      "https",
				Trace:       trace,
				Transport:   eofTransport,
				ZeroTime:    zeroTime,
			}
			httpRequest := &httpRequestFunc{}
			res := httpRequest.Apply(context.Background(), &httpTransport)
			if res.Error != io.EOF {
				t.Fatal("not the error we expected")
			}
			if res.State.HTTPResponse != nil {
				t.Fatal("expected nil request here")
			}
		})

		t.Run("with invalid method", func(t *testing.T) {
			httpTransport := HTTPTransport{
				Address:     "1.2.3.4:567",
				IDGenerator: idGen,
				Logger:      model.DiscardLogger,
				Network:     "tcp",
				Scheme:      "https",
				Trace:       trace,
				Transport:   goodTransport,
				ZeroTime:    zeroTime,
			}
			httpRequest := &httpRequestFunc{
				Method: "€",
			}
			res := httpRequest.Apply(context.Background(), &httpTransport)
			if res.Error == nil || !strings.HasPrefix(res.Error.Error(), "net/http: invalid method") {
				t.Fatal("not the error we expected")
			}
			if res.State.HTTPResponse != nil {
				t.Fatal("expected nil request here")
			}
		})

		t.Run("with port-less address", func(t *testing.T) {
			httpTransport := HTTPTransport{
				Address:     "1.2.3.4",
				IDGenerator: idGen,
				Logger:      model.DiscardLogger,
				Network:     "tcp",
				Scheme:      "https",
				Trace:       trace,
				Transport:   goodTransport,
				ZeroTime:    zeroTime,
			}
			httpRequest := &httpRequestFunc{}
			res := httpRequest.Apply(context.Background(), &httpTransport)
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

		t.Run("with success (https)", func(t *testing.T) {
			httpTransport := HTTPTransport{
				Address:     "1.2.3.4:443",
				IDGenerator: idGen,
				Logger:      model.DiscardLogger,
				Network:     "tcp",
				Scheme:      "https",
				Trace:       trace,
				Transport:   goodTransport,
				ZeroTime:    zeroTime,
			}
			httpRequest := &httpRequestFunc{}
			res := httpRequest.Apply(context.Background(), &httpTransport)
			if res.Error != nil {
				t.Fatal("unexpected error")
			}
			if res.State.HTTPResponse == nil || res.State.HTTPResponse.Status != "expected" {
				t.Fatal("unexpected request")
			}
		})

		t.Run("with success (http)", func(t *testing.T) {
			httpTransport := HTTPTransport{
				Address:     "1.2.3.4:80",
				IDGenerator: idGen,
				Logger:      model.DiscardLogger,
				Network:     "tcp",
				Scheme:      "http",
				Trace:       trace,
				Transport:   goodTransport,
				ZeroTime:    zeroTime,
			}
			httpRequest := &httpRequestFunc{}
			res := httpRequest.Apply(context.Background(), &httpTransport)
			if res.Error != nil {
				t.Fatal("unexpected error")
			}
			if res.State.HTTPResponse == nil || res.State.HTTPResponse.Status != "expected" {
				t.Fatal("unexpected request")
			}
		})

		t.Run("with header options", func(t *testing.T) {
			httpTransport := HTTPTransport{
				Address:     "1.2.3.4:567",
				Domain:      "domain.com",
				IDGenerator: idGen,
				Logger:      model.DiscardLogger,
				Network:     "tcp",
				Scheme:      "https",
				Trace:       trace,
				Transport:   goodTransport,
				ZeroTime:    zeroTime,
			}
			httpRequest := &httpRequestFunc{
				Accept:         "text/html",
				AcceptLanguage: "de",
				Host:           "host",
				Referer:        "https://example.org",
				URLPath:        "/path/to/example",
				UserAgent:      "Mozilla/5.0 Gecko/geckotrail Firefox/firefoxversion",
			}
			res := httpRequest.Apply(context.Background(), &httpTransport)
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
- Get httpTransportTCPFunc
- Get composed function: TCP with HTTP
- Apply httpTransportTCPFunc
*/
func TestHTTPTCP(t *testing.T) {
	t.Run("Get httpTransportTCPFunc", func(t *testing.T) {
		f := HTTPTransportTCP()
		if _, ok := f.(*httpTransportTCPFunc); !ok {
			t.Fatal("unexpected type")
		}
	})

	t.Run("Get composed function: TCP with HTTP", func(t *testing.T) {
		f := HTTPRequestOverTCP()
		if _, ok := f.(*compose2Func[*TCPConnection, *HTTPTransport, *HTTPResponse]); !ok {
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
			Address:     address,
			Conn:        conn,
			IDGenerator: idGen,
			Logger:      model.DiscardLogger,
			Network:     "tcp",
			Trace:       trace,
			ZeroTime:    zeroTime,
		}
		f := httpTransportTCPFunc{}
		res := f.Apply(context.Background(), tcpConn)
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
- Get httpTransportQUICFunc
- Get composed function: QUIC with HTTP
- Apply httpTransportQUICFunc
*/
func TestHTTPQUIC(t *testing.T) {
	t.Run("Get httpTransportQUICFunc", func(t *testing.T) {
		f := HTTPTransportQUIC()
		if _, ok := f.(*httpTransportQUICFunc); !ok {
			t.Fatal("unexpected type")
		}
	})

	t.Run("Get composed function: QUIC with HTTP", func(t *testing.T) {
		f := HTTPRequestOverQUIC()
		if _, ok := f.(*compose2Func[*QUICConnection, *HTTPTransport, *HTTPResponse]); !ok {
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
			Address:     address,
			QUICConn:    conn,
			IDGenerator: idGen,
			Logger:      model.DiscardLogger,
			Network:     "udp",
			Trace:       trace,
			ZeroTime:    zeroTime,
		}
		f := httpTransportQUICFunc{}
		res := f.Apply(context.Background(), quicConn)
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
- Get httpTransportTLSFunc
- Get composed function: TLS with HTTP
- Apply httpTransportTLSFunc
*/
func TestHTTPTLS(t *testing.T) {
	t.Run("Get httpTransportTLSFunc", func(t *testing.T) {
		f := HTTPTransportTLS()
		if _, ok := f.(*httpTransportTLSFunc); !ok {
			t.Fatal("unexpected type")
		}
	})

	t.Run("Get composed function: TLS with HTTP", func(t *testing.T) {
		f := HTTPRequestOverTLS()
		if _, ok := f.(*compose2Func[*TLSConnection, *HTTPTransport, *HTTPResponse]); !ok {
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
			Address:     address,
			Conn:        conn,
			IDGenerator: idGen,
			Logger:      model.DiscardLogger,
			Network:     "tcp",
			Trace:       trace,
			ZeroTime:    zeroTime,
		}
		f := httpTransportTLSFunc{}
		res := f.Apply(context.Background(), tlsConn)
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
