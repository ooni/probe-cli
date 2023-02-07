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

func TestHTTPRequest(t *testing.T) {
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
		t.Fatal("TLSHandshake: unexpected type. Expected: tlsHandshakeFunc")
	}
	if requestFunc.Accept != "text/html" {
		t.Fatalf("HTTPRequest: %s, expected %s, got %s", "Accept", "text/html", requestFunc.Accept)
	}
	if requestFunc.AcceptLanguage != "de" {
		t.Fatalf("HTTPRequest: %s, expected %s, got %s", "AcceptLanguage", "de", requestFunc.AcceptLanguage)
	}
	if requestFunc.Host != "host" {
		t.Fatalf("HTTPRequest: %s, expected %s, got %s", "Host", "host", requestFunc.Host)
	}
	if requestFunc.Method != "PUT" {
		t.Fatalf("HTTPRequest: %s, expected %s, got %s", "Method", "PUT", requestFunc.Method)
	}
	if requestFunc.Referer != "https://example.com/" {
		t.Fatalf("HTTPRequest: %s, expected %s, got %s", "Referer", "https://example.com/", requestFunc.Referer)
	}
	if requestFunc.URLPath != "/path/to/example" {
		t.Fatalf("HTTPRequest: %s, expected %s, got %s", "URLPath", "example/to/path", requestFunc.URLPath)
	}
	if requestFunc.UserAgent != "Mozilla/5.0 Gecko/geckotrail Firefox/firefoxversion" {
		t.Fatalf("HTTPRequest: %s, expected %s, got %s", "UserAgent", "Mozilla/5.0 Gecko/geckotrail Firefox/firefoxversion", requestFunc.UserAgent)
	}
}

func TestApplyHTTP(t *testing.T) {
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
	t.Run("with invalid address", func(t *testing.T) {
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
	t.Run("with https", func(t *testing.T) {
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
	t.Run("with http", func(t *testing.T) {
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
}
