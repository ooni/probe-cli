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

func TestHTTPTransportLogger(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("with failure", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebug: func(message string) {
					count++
				},
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			txp := &httpTransportLogger{
				Logger: lo,
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
			if count < 1 {
				t.Fatal("no logs?!")
			}
		})

		t.Run("we add the host header", func(t *testing.T) {
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
		})

		t.Run("with success", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebug: func(message string) {
					count++
				},
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			txp := &httpTransportLogger{
				Logger: lo,
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
			if count < 1 {
				t.Fatal("no logs?!")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
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
	})
}

func TestHTTPTransportConnectionsCloser(t *testing.T) {
	t.Run("CloseIdleConnections", func(t *testing.T) {
		var (
			calledTxp    bool
			calledDialer bool
			calledTLS    bool
		)
		txp := &httpTransportConnectionsCloser{
			HTTPTransport: &mocks.HTTPTransport{
				MockCloseIdleConnections: func() {
					calledTxp = true
				},
			},
			Dialer: &mocks.Dialer{
				MockCloseIdleConnections: func() {
					calledDialer = true
				},
			},
			TLSDialer: &mocks.TLSDialer{
				MockCloseIdleConnections: func() {
					calledTLS = true
				},
			},
		}
		txp.CloseIdleConnections()
		if !calledDialer || !calledTLS || !calledTxp {
			t.Fatal("not called")
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		expected := errors.New("mocked error")
		txp := &httpTransportConnectionsCloser{
			HTTPTransport: &mocks.HTTPTransport{
				MockRoundTrip: func(req *http.Request) (*http.Response, error) {
					return nil, expected
				},
			},
		}
		client := &http.Client{Transport: txp}
		resp, err := client.Get("https://www.google.com")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("unexpected resp")
		}
	})
}

func TestNewHTTPTransport(t *testing.T) {
	t.Run("works as intended with failing dialer", func(t *testing.T) {
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
	})

	t.Run("creates the correct type chain", func(t *testing.T) {
		d := &mocks.Dialer{}
		td := &mocks.TLSDialer{}
		txp := NewHTTPTransport(log.Log, d, td)
		logger := txp.(*httpTransportLogger)
		if logger.Logger != log.Log {
			t.Fatal("invalid logger")
		}
		connectionsCloser := logger.HTTPTransport.(*httpTransportConnectionsCloser)
		withReadTimeout := connectionsCloser.Dialer.(*httpDialerWithReadTimeout)
		if withReadTimeout.Dialer != d {
			t.Fatal("invalid dialer")
		}
		tlsWithReadTimeout := connectionsCloser.TLSDialer.(*httpTLSDialerWithReadTimeout)
		if tlsWithReadTimeout.TLSDialer != td {
			t.Fatal("invalid tls dialer")
		}
		stdlib := connectionsCloser.HTTPTransport.(*oohttp.StdlibTransport)
		if !stdlib.Transport.ForceAttemptHTTP2 {
			t.Fatal("invalid ForceAttemptHTTP2")
		}
		if !stdlib.Transport.DisableCompression {
			t.Fatal("invalid DisableCompression")
		}
		if stdlib.Transport.MaxConnsPerHost != 1 {
			t.Fatal("invalid MaxConnPerHost")
		}
		if stdlib.Transport.DialTLSContext == nil {
			t.Fatal("invalid DialTLSContext")
		}
		if stdlib.Transport.DialContext == nil {
			t.Fatal("invalid DialContext")
		}
	})
}

func TestHTTPDialerWithReadTimeout(t *testing.T) {
	t.Run("on success", func(t *testing.T) {
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
	})

	t.Run("on failure", func(t *testing.T) {
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
	})
}

func TestHTTPTLSDialerWithReadTimeout(t *testing.T) {
	t.Run("on success", func(t *testing.T) {
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
	})

	t.Run("on failure", func(t *testing.T) {
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
	})

	t.Run("with invalid conn type", func(t *testing.T) {
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
	})
}
