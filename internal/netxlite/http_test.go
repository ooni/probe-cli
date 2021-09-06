package netxlite

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestHTTPTransportLoggerFailure(t *testing.T) {
	txp := &httpTransportLogger{
		Logger: log.Log,
		HTTPTransport: &mocks.HTTPTransport{
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
	txp := &httpTransportLogger{
		Logger: log.Log,
		HTTPTransport: &mocks.HTTPTransport{
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
	txp := &httpTransportLogger{
		Logger: log.Log,
		HTTPTransport: &mocks.HTTPTransport{
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
	txp := &httpTransportLogger{
		HTTPTransport: &mocks.HTTPTransport{
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

func TestHTTPTransportWorks(t *testing.T) {
	d := NewDialerWithResolver(log.Log, NewResolverSystem(log.Log))
	td := NewTLSDialer(d, NewTLSHandshakerStdlib(log.Log))
	txp := NewHTTPTransport(log.Log, d, td)
	client := &http.Client{Transport: txp}
	defer client.CloseIdleConnections()
	resp, err := client.Get("https://www.google.com/robots.txt")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestHTTPTransportWithFailingDialer(t *testing.T) {
	called := &atomicx.Int64{}
	expected := errors.New("mocked error")
	d := &dialerResolver{
		Dialer: &mocks.Dialer{
			MockDialContext: func(ctx context.Context,
				network, address string) (net.Conn, error) {
				return nil, expected
			},
			MockCloseIdleConnections: func() {
				called.Add(1)
			},
		},
		Resolver: NewResolverSystem(log.Log),
	}
	td := NewTLSDialer(d, NewTLSHandshakerStdlib(log.Log))
	txp := NewHTTPTransport(log.Log, d, td)
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com/robots.txt")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if resp != nil {
		t.Fatal("expected non-nil response here")
	}
	client.CloseIdleConnections()
	if called.Load() < 1 {
		t.Fatal("did not propagate CloseIdleConnections")
	}
}

func TestNewHTTPTransport(t *testing.T) {
	d := &mocks.Dialer{}
	td := &mocks.TLSDialer{}
	txp := NewHTTPTransport(log.Log, d, td)
	logtxp, okay := txp.(*httpTransportLogger)
	if !okay {
		t.Fatal("invalid type")
	}
	if logtxp.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	txpcc, okay := logtxp.HTTPTransport.(*httpTransportConnectionsCloser)
	if !okay {
		t.Fatal("invalid type")
	}
	udt, okay := txpcc.Dialer.(*httpDialerWithReadTimeout)
	if !okay {
		t.Fatal("invalid type")
	}
	if udt.Dialer != d {
		t.Fatal("invalid dialer")
	}
	utdt, okay := txpcc.TLSDialer.(*httpTLSDialerWithReadTimeout)
	if !okay {
		t.Fatal("invalid type")
	}
	if utdt.TLSDialer != td {
		t.Fatal("invalid tls dialer")
	}
	stdwtxp, okay := txpcc.HTTPTransport.(*oohttp.StdlibTransport)
	if !okay {
		t.Fatal("invalid type")
	}
	if !stdwtxp.Transport.ForceAttemptHTTP2 {
		t.Fatal("invalid ForceAttemptHTTP2")
	}
	if !stdwtxp.Transport.DisableCompression {
		t.Fatal("invalid DisableCompression")
	}
	if stdwtxp.Transport.MaxConnsPerHost != 1 {
		t.Fatal("invalid MaxConnPerHost")
	}
	if stdwtxp.Transport.DialTLSContext == nil {
		t.Fatal("invalid DialTLSContext")
	}
	if stdwtxp.Transport.DialContext == nil {
		t.Fatal("invalid DialContext")
	}
}

func TestHTTPDialerWithReadTimeout(t *testing.T) {
	var (
		calledWithZeroTime    bool
		calledWithNonZeroTime bool
	)
	origConn := &mocks.Conn{
		MockSetReadDeadline: func(t time.Time) error {
			switch t.IsZero() {
			case true:
				calledWithZeroTime = true
			case false:
				calledWithNonZeroTime = true
			}
			return nil
		},
		MockRead: func(b []byte) (int, error) {
			return 0, io.EOF
		},
	}
	d := &httpDialerWithReadTimeout{
		Dialer: &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return origConn, nil
			},
		},
	}
	ctx := context.Background()
	conn, err := d.DialContext(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, okay := conn.(*httpConnWithReadTimeout); !okay {
		t.Fatal("invalid conn type")
	}
	if conn.(*httpConnWithReadTimeout).Conn != origConn {
		t.Fatal("invalid origin conn")
	}
	b := make([]byte, 1024)
	count, err := conn.Read(b)
	if !errors.Is(err, io.EOF) {
		t.Fatal("invalid error")
	}
	if count != 0 {
		t.Fatal("invalid count")
	}
	if !calledWithZeroTime || !calledWithNonZeroTime {
		t.Fatal("not called")
	}
}

func TestHTTPTLSDialerWithReadTimeout(t *testing.T) {
	var (
		calledWithZeroTime    bool
		calledWithNonZeroTime bool
	)
	origConn := &mocks.TLSConn{
		Conn: mocks.Conn{
			MockSetReadDeadline: func(t time.Time) error {
				switch t.IsZero() {
				case true:
					calledWithZeroTime = true
				case false:
					calledWithNonZeroTime = true
				}
				return nil
			},
			MockRead: func(b []byte) (int, error) {
				return 0, io.EOF
			},
		},
	}
	d := &httpTLSDialerWithReadTimeout{
		TLSDialer: &mocks.TLSDialer{
			MockDialTLSContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return origConn, nil
			},
		},
	}
	ctx := context.Background()
	conn, err := d.DialTLSContext(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, okay := conn.(*httpTLSConnWithReadTimeout); !okay {
		t.Fatal("invalid conn type")
	}
	if conn.(*httpTLSConnWithReadTimeout).TLSConn != origConn {
		t.Fatal("invalid origin conn")
	}
	b := make([]byte, 1024)
	count, err := conn.Read(b)
	if !errors.Is(err, io.EOF) {
		t.Fatal("invalid error")
	}
	if count != 0 {
		t.Fatal("invalid count")
	}
	if !calledWithZeroTime || !calledWithNonZeroTime {
		t.Fatal("not called")
	}
}

func TestHTTPDialerWithReadTimeoutDialingFailure(t *testing.T) {
	expected := errors.New("mocked error")
	d := &httpDialerWithReadTimeout{
		Dialer: &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return nil, expected
			},
		},
	}
	conn, err := d.DialContext(context.Background(), "", "")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestHTTPTLSDialerWithReadTimeoutDialingFailure(t *testing.T) {
	expected := errors.New("mocked error")
	d := &httpTLSDialerWithReadTimeout{
		TLSDialer: &mocks.TLSDialer{
			MockDialTLSContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return nil, expected
			},
		},
	}
	conn, err := d.DialTLSContext(context.Background(), "", "")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestHTTPTLSDialerWithInvalidConnType(t *testing.T) {
	var called bool
	d := &httpTLSDialerWithReadTimeout{
		TLSDialer: &mocks.TLSDialer{
			MockDialTLSContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return &mocks.Conn{
					MockClose: func() error {
						called = true
						return nil
					},
				}, nil
			},
		},
	}
	conn, err := d.DialTLSContext(context.Background(), "", "")
	if !errors.Is(err, ErrNotTLSConn) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
	if !called {
		t.Fatal("not called")
	}
}
