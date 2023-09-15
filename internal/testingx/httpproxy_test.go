package testingx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingproxy"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestHTTPProxyHandler(t *testing.T) {
	for _, testCase := range testingproxy.AllTestCases {
		t.Run(testCase.Name(), func(t *testing.T) {
			short := testCase.Short()
			if !short && testing.Short() {
				t.Skip("skip test in short mode")
			}
			testCase.Run(t)
		})
	}

	t.Run("rejects requests without a host header", func(t *testing.T) {
		rr := httptest.NewRecorder()
		netx := &netxlite.Netx{Underlying: &mocks.UnderlyingNetwork{
			// all nil: panic if we hit the network
		}}
		handler := testingx.NewHTTPProxyHandler(log.Log, netx)
		req := &http.Request{
			Method: http.MethodGet,
			Host:   "", // explicitly empty
		}
		handler.ServeHTTP(rr, req)
		res := rr.Result()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code", res.StatusCode)
		}
	})

	t.Run("rejects requests with a via header", func(t *testing.T) {
		rr := httptest.NewRecorder()
		netx := &netxlite.Netx{Underlying: &mocks.UnderlyingNetwork{
			// all nil: panic if we hit the network
		}}
		handler := testingx.NewHTTPProxyHandler(log.Log, netx)
		req := &http.Request{
			Method: http.MethodGet,
			Host:   "www.example.com",
			Header: http.Header{
				"Via": {"antani/0.1.0"},
			},
		}
		handler.ServeHTTP(rr, req)
		res := rr.Result()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code", res.StatusCode)
		}
	})

	t.Run("rejects requests with a POST method", func(t *testing.T) {
		rr := httptest.NewRecorder()
		netx := &netxlite.Netx{Underlying: &mocks.UnderlyingNetwork{
			// all nil: panic if we hit the network
		}}
		handler := testingx.NewHTTPProxyHandler(log.Log, netx)
		req := &http.Request{
			Method: http.MethodPost,
			Host:   "www.example.com",
			Header: http.Header{},
		}
		handler.ServeHTTP(rr, req)
		res := rr.Result()
		if res.StatusCode != http.StatusNotImplemented {
			t.Fatal("unexpected status code", res.StatusCode)
		}
	})

	t.Run("returns 502 when the round trip fails", func(t *testing.T) {
		t.Run("with a GET request", func(t *testing.T) {
			rr := httptest.NewRecorder()
			netx := &netxlite.Netx{Underlying: &mocks.UnderlyingNetwork{
				MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
					return nil, "", errors.New("mocked error")
				},
				MockGetaddrinfoResolverNetwork: func() string {
					return "antani"
				},
			}}
			handler := testingx.NewHTTPProxyHandler(log.Log, netx)
			req := &http.Request{
				Method: http.MethodGet,
				Host:   "www.example.com",
				Header: http.Header{},
				URL:    &url.URL{},
			}
			handler.ServeHTTP(rr, req)
			res := rr.Result()
			if res.StatusCode != http.StatusBadGateway {
				t.Fatal("unexpected status code", res.StatusCode)
			}
		})

		t.Run("with a CONNECT request", func(t *testing.T) {
			rr := httptest.NewRecorder()
			netx := &netxlite.Netx{Underlying: &mocks.UnderlyingNetwork{
				MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
					return nil, "", errors.New("mocked error")
				},
				MockGetaddrinfoResolverNetwork: func() string {
					return "antani"
				},
			}}
			handler := testingx.NewHTTPProxyHandler(log.Log, netx)
			req := &http.Request{
				Method: http.MethodConnect,
				Host:   "www.example.com:443",
				Header: http.Header{},
				URL:    &url.URL{},
			}
			handler.ServeHTTP(rr, req)
			res := rr.Result()
			if res.StatusCode != http.StatusBadGateway {
				t.Fatal("unexpected status code", res.StatusCode)
			}
		})
	})
}
