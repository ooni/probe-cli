package httptransport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
	"testing"

	"golang.org/x/net/http2"
)

type MockTLSDialer struct {
	err error
}

func (d MockTLSDialer) DialTLSContext(ctx context.Context, network string, address string) (net.Conn, error) {
	return nil, d.err
}

var mocktlsdialer MockTLSDialer = MockTLSDialer{err: errors.New("mock error")}
var txp *http.Transport = http.DefaultTransport.(*http.Transport).Clone()

func TestGetTransportHTTPS(t *testing.T) {
	rt := roundTripper{underlyingTransport: txp, DialTLS: mocktlsdialer.DialTLSContext, tlsconfig: &tls.Config{}}
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "www.google.com",
		},
		Header: http.Header{},
	}
	err := rt.getTransport(req)
	if err != nil {
		t.Fatal("unexpected failure", err)
	}
	transport := rt.transport
	if transport == nil {
		t.Fatal("unexpected nil transport")
	}
	if _, ok := transport.(*http2.Transport); !ok {
		t.Fatal("unexpected transport type")
	}
}

func TestGetTransportHTTP(t *testing.T) {
	rt := roundTripper{underlyingTransport: txp, DialTLS: mocktlsdialer.DialTLSContext, tlsconfig: &tls.Config{}}
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   "www.google.com",
		},
		Header: http.Header{},
	}
	err := rt.getTransport(req)
	if err != nil {
		t.Fatal("unexpected failure")
	}
	transport := rt.transport
	if transport == nil {
		t.Fatal("unexpected nil transport")
	}
	if _, ok := transport.(*http.Transport); !ok {
		t.Fatal("unexpected transport type")
	}
}

func TestGetTransportHTTP1TLS(t *testing.T) {
	rt := roundTripper{underlyingTransport: txp, DialTLS: mocktlsdialer.DialTLSContext, tlsconfig: &tls.Config{}}
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "geoip.ubuntu.com",
		},
		Header: http.Header{},
	}
	err := rt.getTransport(req)
	if err != nil {
		t.Fatal("unexpected failure")
	}
	transport := rt.transport
	if transport == nil {
		t.Fatal("unexpected nil transport")
	}
	if _, ok := transport.(*http.Transport); !ok {
		t.Fatal("unexpected transport type")
	}
}

func TestGetTransportAlreadySet(t *testing.T) {
	noerrorDialer := MockTLSDialer{err: nil}
	rt := roundTripper{underlyingTransport: txp, DialTLS: noerrorDialer.DialTLSContext, tlsconfig: &tls.Config{}}
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "www.google.com:443",
		},
		Header: http.Header{},
	}
	rt.transport = txp
	err := rt.getTransport(req)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err.Error() != "dialTLS returned no error when determining transport" {
		t.Fatal("unexpected error type")
	}
}

func TestGetTransportInvalidScheme(t *testing.T) {
	rt := roundTripper{underlyingTransport: txp, DialTLS: mocktlsdialer.DialTLSContext, tlsconfig: &tls.Config{}}
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "x",
			Host:   "www.google.com",
		},
		Header: http.Header{},
	}
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected an error here")
	}
	transport := rt.transport
	if transport != nil {
		t.Fatal("unexpected non-nil transport")
	}
}

func TestRoundTripSuccess(t *testing.T) {
	expected := errors.New("expected error")
	mocktlsdialer := MockTLSDialer{err: expected}
	txp := http.DefaultTransport.(*http.Transport).Clone()
	rt := newRoundtripper(txp, mocktlsdialer, nil)
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "www.google.com",
		},
		Header: http.Header{},
	}
	resp, err := rt.RoundTrip(req)
	if err != expected {
		t.Fatal("unexpected error", err)
	}
	if resp != nil {
		t.Fatal("unexpected non-nil response")
	}

}

func TestConnectFail(t *testing.T) {
	rt := newRoundtripper(txp, mocktlsdialer, nil)
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "https",
			Host:   "a.b.c.d:0",
		},
		Header: http.Header{},
	}
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error an error here")
	}
}

func TestHandshakeFail(t *testing.T) {
	rt := roundTripper{underlyingTransport: txp, DialTLS: mocktlsdialer.DialTLSContext, tlsconfig: &tls.Config{ServerName: "mockname"}}
	_, err := rt.dialTLSContext(context.Background(), "tcp", "google.com:443")
	if err == nil {
		t.Fatal("expected error an error here")
	}
	var hostnameErr x509.HostnameError
	if !errors.As(err, &hostnameErr) {
		t.Fatal("unexpected error type")
	}
}

func TestCanceled(t *testing.T) {
	rt := roundTripper{underlyingTransport: txp, DialTLS: mocktlsdialer.DialTLSContext}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := rt.dialTLSContext(ctx, "tcp", "www.google.com:443")
	if err == nil {
		t.Fatal("expected error an error here")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatal("unexpected error type")
	}
}
