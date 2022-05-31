package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestNewQUICListener(t *testing.T) {
	ql := NewQUICListener()
	qew := ql.(*quicListenerErrWrapper)
	_ = qew.QUICListener.(*quicListenerStdlib)
}

func TestNewQUICDialer(t *testing.T) {
	ql := NewQUICListener()
	dlr := NewQUICDialerWithoutResolver(ql, log.Log)
	logger := dlr.(*quicDialerLogger)
	if logger.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	resolver := logger.Dialer.(*quicDialerResolver)
	if _, okay := resolver.Resolver.(*NullResolver); !okay {
		t.Fatal("invalid resolver type")
	}
	logger = resolver.Dialer.(*quicDialerLogger)
	if logger.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	errWrapper := logger.Dialer.(*quicDialerErrWrapper)
	handshakeCompleter := errWrapper.QUICDialer.(*quicDialerHandshakeCompleter)
	base := handshakeCompleter.Dialer.(*quicDialerQUICGo)
	if base.QUICListener != ql {
		t.Fatal("invalid quic listener")
	}
}

func TestParseUDPAddr(t *testing.T) {
	t.Run("cannot split host and port", func(t *testing.T) {
		addr, err := ParseUDPAddr("1.2.3.4")
		if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
			t.Fatal("unexpected error", err)
		}
		if addr != nil {
			t.Fatal("expected nil addr")
		}
	})

	t.Run("with invalid IP addr", func(t *testing.T) {
		addr, err := ParseUDPAddr("www.google.com:80")
		if !errors.Is(err, ErrInvalidIP) {
			t.Fatal("unexpected error", err)
		}
		if addr != nil {
			t.Fatal("expected nil addr")
		}
	})

	t.Run("with invalid port", func(t *testing.T) {
		addr, err := ParseUDPAddr("8.8.8.8:www")
		if err == nil || !strings.HasSuffix(err.Error(), "invalid syntax") {
			t.Fatal("unexpected error", err)
		}
		if addr != nil {
			t.Fatal("expected nil addr")
		}
	})

	t.Run("with valid input", func(t *testing.T) {
		addr, err := ParseUDPAddr("8.8.8.8:80")
		if err != nil {
			t.Fatal(err)
		}
		if addr.IP.String() != "8.8.8.8" {
			t.Fatal("invalid IP")
		}
		if addr.Port != 80 {
			t.Fatal("invalid port")
		}
		if addr.Zone != "" {
			t.Fatal("invalid zone")
		}
	})
}

func TestQUICDialerQUICGo(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		t.Run("cannot split host port", func(t *testing.T) {
			tlsConfig := &tls.Config{
				ServerName: "www.google.com",
			}
			systemdialer := quicDialerQUICGo{
				QUICListener: &quicListenerStdlib{},
			}
			defer systemdialer.CloseIdleConnections() // just to see it running
			ctx := context.Background()
			qconn, err := systemdialer.DialContext(
				ctx, "udp", "a.b.c.d", tlsConfig, &quic.Config{})
			if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
				t.Fatal("not the error we expected", err)
			}
			if qconn != nil {
				t.Fatal("expected nil connection here")
			}
		})

		t.Run("with invalid port", func(t *testing.T) {
			tlsConfig := &tls.Config{
				ServerName: "www.google.com",
			}
			systemdialer := quicDialerQUICGo{
				QUICListener: &quicListenerStdlib{},
			}
			ctx := context.Background()
			qconn, err := systemdialer.DialContext(
				ctx, "udp", "8.8.4.4:xyz", tlsConfig, &quic.Config{})
			if err == nil || !strings.HasSuffix(err.Error(), "invalid syntax") {
				t.Fatal("not the error we expected", err)
			}
			if qconn != nil {
				t.Fatal("expected nil connection here")
			}
		})

		t.Run("with invalid IP", func(t *testing.T) {
			tlsConfig := &tls.Config{
				ServerName: "www.google.com",
			}
			systemdialer := quicDialerQUICGo{
				QUICListener: &quicListenerStdlib{},
			}
			ctx := context.Background()
			qconn, err := systemdialer.DialContext(
				ctx, "udp", "a.b.c.d:0", tlsConfig, &quic.Config{})
			if !errors.Is(err, ErrInvalidIP) {
				t.Fatal("not the error we expected", err)
			}
			if qconn != nil {
				t.Fatal("expected nil connection here")
			}
		})

		t.Run("with listen error", func(t *testing.T) {
			expected := errors.New("mocked error")
			tlsConfig := &tls.Config{
				ServerName: "www.google.com",
			}
			systemdialer := quicDialerQUICGo{
				QUICListener: &mocks.QUICListener{
					MockListen: func(addr *net.UDPAddr) (model.UDPLikeConn, error) {
						return nil, expected
					},
				},
			}
			ctx := context.Background()
			qconn, err := systemdialer.DialContext(
				ctx, "udp", "8.8.8.8:443", tlsConfig, &quic.Config{})
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected", err)
			}
			if qconn != nil {
				t.Fatal("expected nil connection here")
			}
		})

		t.Run("with handshake failure", func(t *testing.T) {
			tlsConfig := &tls.Config{
				ServerName: "dns.google",
			}
			systemdialer := quicDialerQUICGo{
				QUICListener: &quicListenerStdlib{},
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // fail immediately
			qconn, err := systemdialer.DialContext(
				ctx, "udp", "8.8.8.8:443", tlsConfig, &quic.Config{})
			if !errors.Is(err, context.Canceled) {
				t.Fatal("not the error we expected", err)
			}
			if qconn != nil {
				log.Fatal("expected nil connection here")
			}
		})

		t.Run("TLS defaults for web", func(t *testing.T) {
			expected := errors.New("mocked error")
			var gotTLSConfig *tls.Config
			tlsConfig := &tls.Config{
				ServerName: "dns.google",
			}
			systemdialer := quicDialerQUICGo{
				QUICListener: &quicListenerStdlib{},
				mockDialEarlyContext: func(ctx context.Context, pconn net.PacketConn,
					remoteAddr net.Addr, host string, tlsConfig *tls.Config,
					quicConfig *quic.Config) (quic.EarlyConnection, error) {
					gotTLSConfig = tlsConfig
					return nil, expected
				},
			}
			ctx := context.Background()
			qconn, err := systemdialer.DialContext(
				ctx, "udp", "8.8.8.8:443", tlsConfig, &quic.Config{})
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected", err)
			}
			if qconn != nil {
				t.Fatal("expected nil connection here")
			}
			if tlsConfig.RootCAs != nil {
				t.Fatal("tlsConfig.RootCAs should not have been changed")
			}
			if gotTLSConfig.RootCAs != defaultCertPool {
				t.Fatal("invalid gotTLSConfig.RootCAs")
			}
			if tlsConfig.NextProtos != nil {
				t.Fatal("tlsConfig.NextProtos should not have been changed")
			}
			if diff := cmp.Diff(gotTLSConfig.NextProtos, []string{"h3"}); diff != "" {
				t.Fatal("invalid gotTLSConfig.NextProtos", diff)
			}
			if tlsConfig.ServerName != gotTLSConfig.ServerName {
				t.Fatal("the ServerName field must match")
			}
		})

		t.Run("TLS defaults for DoQ", func(t *testing.T) {
			expected := errors.New("mocked error")
			var gotTLSConfig *tls.Config
			tlsConfig := &tls.Config{
				ServerName: "dns.google",
			}
			systemdialer := quicDialerQUICGo{
				QUICListener: &quicListenerStdlib{},
				mockDialEarlyContext: func(ctx context.Context, pconn net.PacketConn,
					remoteAddr net.Addr, host string, tlsConfig *tls.Config,
					quicConfig *quic.Config) (quic.EarlyConnection, error) {
					gotTLSConfig = tlsConfig
					return nil, expected
				},
			}
			ctx := context.Background()
			qconn, err := systemdialer.DialContext(
				ctx, "udp", "8.8.8.8:8853", tlsConfig, &quic.Config{})
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected", err)
			}
			if qconn != nil {
				t.Fatal("expected nil connection here")
			}
			if tlsConfig.RootCAs != nil {
				t.Fatal("tlsConfig.RootCAs should not have been changed")
			}
			if gotTLSConfig.RootCAs != defaultCertPool {
				t.Fatal("invalid gotTLSConfig.RootCAs")
			}
			if tlsConfig.NextProtos != nil {
				t.Fatal("tlsConfig.NextProtos should not have been changed")
			}
			if diff := cmp.Diff(gotTLSConfig.NextProtos, []string{"dq"}); diff != "" {
				t.Fatal("invalid gotTLSConfig.NextProtos", diff)
			}
			if tlsConfig.ServerName != gotTLSConfig.ServerName {
				t.Fatal("the ServerName field must match")
			}
		})
	})
}

func TestQUICDialerHandshakeCompleter(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		t.Run("in case of failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			d := &quicDialerHandshakeCompleter{
				Dialer: &mocks.QUICDialer{
					MockDialContext: func(ctx context.Context, network, address string,
						tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
						return nil, expected
					},
				},
			}
			ctx := context.Background()
			conn, err := d.DialContext(ctx, "udp", "8.8.8.8:443", &tls.Config{}, &quic.Config{})
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})

		t.Run("in case of context cancellation", func(t *testing.T) {
			handshakeCtx, handshakeCancel := context.WithCancel(context.Background())
			defer handshakeCancel()
			ctx, cancel := context.WithCancel(context.Background())
			var called bool
			expected := &mocks.QUICEarlyConnection{
				MockHandshakeComplete: func() context.Context {
					cancel()
					return handshakeCtx
				},
				MockCloseWithError: func(code quic.ApplicationErrorCode, reason string) error {
					called = true
					return nil
				},
			}
			d := &quicDialerHandshakeCompleter{
				Dialer: &mocks.QUICDialer{
					MockDialContext: func(ctx context.Context, network, address string,
						tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
						return expected, nil
					},
				},
			}
			conn, err := d.DialContext(ctx, "udp", "8.8.8.8:443", &tls.Config{}, &quic.Config{})
			if !errors.Is(err, context.Canceled) {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
			if !called {
				t.Fatal("not called")
			}
		})

		t.Run("in case of success", func(t *testing.T) {
			handshakeCtx, handshakeCancel := context.WithCancel(context.Background())
			defer handshakeCancel()
			expected := &mocks.QUICEarlyConnection{
				MockHandshakeComplete: func() context.Context {
					handshakeCancel()
					return handshakeCtx
				},
			}
			d := &quicDialerHandshakeCompleter{
				Dialer: &mocks.QUICDialer{
					MockDialContext: func(ctx context.Context, network, address string,
						tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
						return expected, nil
					},
				},
			}
			conn, err := d.DialContext(
				context.Background(), "udp", "8.8.8.8:443", &tls.Config{}, &quic.Config{})
			if err != nil {
				t.Fatal(err)
			}
			if conn == nil {
				t.Fatal("expected non-nil conn")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var forDialer bool
		d := &quicDialerHandshakeCompleter{
			Dialer: &mocks.QUICDialer{
				MockCloseIdleConnections: func() {
					forDialer = true
				},
			},
		}
		d.CloseIdleConnections()
		if !forDialer {
			t.Fatal("not called")
		}
	})
}

func TestQUICDialerResolver(t *testing.T) {
	t.Run("CloseIdleConnections", func(t *testing.T) {
		var (
			forDialer   bool
			forResolver bool
		)
		d := &quicDialerResolver{
			Dialer: &mocks.QUICDialer{
				MockCloseIdleConnections: func() {
					forDialer = true
				},
			},
			Resolver: &mocks.Resolver{
				MockCloseIdleConnections: func() {
					forResolver = true
				},
			},
		}
		d.CloseIdleConnections()
		if !forDialer || !forResolver {
			t.Fatal("not called")
		}
	})

	t.Run("DialContext", func(t *testing.T) {
		t.Run("with missing port", func(t *testing.T) {
			tlsConfig := &tls.Config{}
			dialer := &quicDialerResolver{
				Resolver: NewResolverStdlib(log.Log),
				Dialer:   &quicDialerQUICGo{}}
			qconn, err := dialer.DialContext(
				context.Background(), "udp", "www.google.com",
				tlsConfig, &quic.Config{})
			if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
				t.Fatal("not the error we expected")
			}
			if qconn != nil {
				t.Fatal("expected a nil connection here")
			}
		})

		t.Run("with lookup host failure", func(t *testing.T) {
			tlsConfig := &tls.Config{}
			expected := errors.New("mocked error")
			dialer := &quicDialerResolver{Resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, expected
				},
			}}
			qconn, err := dialer.DialContext(
				context.Background(), "udp", "dns.google.com:853",
				tlsConfig, &quic.Config{})
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected")
			}
			if qconn != nil {
				t.Fatal("expected nil connection")
			}
		})

		t.Run("with invalid, non-numeric port)", func(t *testing.T) {
			// This test allows us to check for the case where every attempt
			// to establish a connection leads to a failure
			tlsConf := &tls.Config{}
			dialer := &quicDialerResolver{
				Resolver: NewResolverStdlib(log.Log),
				Dialer: &quicDialerQUICGo{
					QUICListener: &quicListenerStdlib{},
				}}
			qconn, err := dialer.DialContext(
				context.Background(), "udp", "8.8.4.4:x",
				tlsConf, &quic.Config{})
			if err == nil {
				t.Fatal("expected an error here")
			}
			if !strings.HasSuffix(err.Error(), "invalid syntax") {
				t.Fatal("not the error we expected", err)
			}
			if qconn != nil {
				t.Fatal("expected nil connection")
			}
		})

		t.Run("we apply TLS defaults", func(t *testing.T) {
			expected := errors.New("mocked error")
			var gotTLSConfig *tls.Config
			tlsConfig := &tls.Config{}
			dialer := &quicDialerResolver{
				Resolver: NewResolverStdlib(log.Log),
				Dialer: &mocks.QUICDialer{
					MockDialContext: func(ctx context.Context, network, address string,
						tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
						gotTLSConfig = tlsConfig
						return nil, expected
					},
				}}
			qconn, err := dialer.DialContext(
				context.Background(), "udp", "8.8.4.4:443",
				tlsConfig, &quic.Config{})
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected", err)
			}
			if qconn != nil {
				t.Fatal("expected nil connection here")
			}
			if tlsConfig.ServerName != "" {
				t.Fatal("should not have changed tlsConfig.ServerName")
			}
			if gotTLSConfig.ServerName != "8.8.4.4" {
				t.Fatal("gotTLSConfig.ServerName has not been set")
			}
		})
	})

	t.Run("lookup host with address", func(t *testing.T) {
		dialer := &quicDialerResolver{Resolver: &mocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				// We should not arrive here and call this function but if we do then
				// there is going to be an error that fails this test.
				return nil, errors.New("mocked error")
			},
		}}
		addrs, err := dialer.lookupHost(context.Background(), "1.1.1.1")
		if err != nil {
			t.Fatal(err)
		}
		if len(addrs) != 1 || addrs[0] != "1.1.1.1" {
			t.Fatal("not the result we expected")
		}
	})
}

func TestQUICLoggerDialer(t *testing.T) {
	t.Run("CloseIdleConnections", func(t *testing.T) {
		var forDialer bool
		d := &quicDialerLogger{
			Dialer: &mocks.QUICDialer{
				MockCloseIdleConnections: func() {
					forDialer = true
				},
			},
		}
		d.CloseIdleConnections()
		if !forDialer {
			t.Fatal("not called")
		}
	})

	t.Run("DialContext", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			var called int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					called++
				},
			}
			d := &quicDialerLogger{
				Dialer: &mocks.QUICDialer{
					MockDialContext: func(ctx context.Context, network string,
						address string, tlsConfig *tls.Config,
						quicConfig *quic.Config) (quic.EarlyConnection, error) {
						return &mocks.QUICEarlyConnection{
							MockCloseWithError: func(
								code quic.ApplicationErrorCode, reason string) error {
								return nil
							},
						}, nil
					},
				},
				Logger: lo,
			}
			ctx := context.Background()
			tlsConfig := &tls.Config{}
			quicConfig := &quic.Config{}
			qconn, err := d.DialContext(ctx, "udp", "8.8.8.8:443", tlsConfig, quicConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err := qconn.CloseWithError(0, ""); err != nil {
				t.Fatal(err)
			}
			if called != 2 {
				t.Fatal("invalid number of calls")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			var called int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					called++
				},
			}
			expected := errors.New("mocked error")
			d := &quicDialerLogger{
				Dialer: &mocks.QUICDialer{
					MockDialContext: func(ctx context.Context, network string,
						address string, tlsConfig *tls.Config,
						quicConfig *quic.Config) (quic.EarlyConnection, error) {
						return nil, expected
					},
				},
				Logger: lo,
			}
			ctx := context.Background()
			tlsConfig := &tls.Config{}
			quicConfig := &quic.Config{}
			qconn, err := d.DialContext(ctx, "udp", "8.8.8.8:443", tlsConfig, quicConfig)
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected", err)
			}
			if qconn != nil {
				t.Fatal("expected nil connection")
			}
			if called != 2 {
				t.Fatal("invalid number of calls")
			}
		})
	})
}

func TestNewSingleUseQUICDialer(t *testing.T) {
	qconn := &mocks.QUICEarlyConnection{}
	qd := NewSingleUseQUICDialer(qconn)
	defer qd.CloseIdleConnections()
	outconn, err := qd.DialContext(
		context.Background(), "", "", &tls.Config{}, &quic.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if qconn != outconn {
		t.Fatal("invalid outconn")
	}
	for i := 0; i < 4; i++ {
		outconn, err = qd.DialContext(
			context.Background(), "", "", &tls.Config{}, &quic.Config{})
		if !errors.Is(err, ErrNoConnReuse) {
			t.Fatal("not the error we expected", err)
		}
		if outconn != nil {
			t.Fatal("expected nil outconn here")
		}
	}
}

func TestQUICListenerErrWrapper(t *testing.T) {
	t.Run("Listen", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			expectedConn := &mocks.UDPLikeConn{}
			ql := &quicListenerErrWrapper{
				QUICListener: &mocks.QUICListener{
					MockListen: func(addr *net.UDPAddr) (model.UDPLikeConn, error) {
						return expectedConn, nil
					},
				},
			}
			conn, err := ql.Listen(&net.UDPAddr{})
			if err != nil {
				t.Fatal(err)
			}
			ewconn := conn.(*quicErrWrapperUDPLikeConn)
			if ewconn.UDPLikeConn != expectedConn {
				t.Fatal("unexpected conn")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expectedErr := io.EOF
			ql := &quicListenerErrWrapper{
				QUICListener: &mocks.QUICListener{
					MockListen: func(addr *net.UDPAddr) (model.UDPLikeConn, error) {
						return nil, expectedErr
					},
				},
			}
			conn, err := ql.Listen(&net.UDPAddr{})
			if err == nil || err.Error() != FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})
	})
}

func TestQUICErrWrapperUDPLikeConn(t *testing.T) {
	t.Run("ReadFrom", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			expectedAddr := &net.UDPAddr{}
			p := make([]byte, 128)
			conn := &quicErrWrapperUDPLikeConn{
				UDPLikeConn: &mocks.UDPLikeConn{
					MockReadFrom: func(p []byte) (n int, addr net.Addr, err error) {
						return len(p), expectedAddr, nil
					},
				},
			}
			count, addr, err := conn.ReadFrom(p)
			if err != nil {
				t.Fatal(err)
			}
			if count != len(p) {
				t.Fatal("unexpected count")
			}
			if addr != expectedAddr {
				t.Fatal("unexpected addr")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			p := make([]byte, 128)
			expectedErr := io.EOF
			conn := &quicErrWrapperUDPLikeConn{
				UDPLikeConn: &mocks.UDPLikeConn{
					MockReadFrom: func(p []byte) (n int, addr net.Addr, err error) {
						return 0, nil, expectedErr
					},
				},
			}
			count, addr, err := conn.ReadFrom(p)
			if err == nil || err.Error() != FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if count != 0 {
				t.Fatal("unexpected count")
			}
			if addr != nil {
				t.Fatal("unexpected addr")
			}
		})
	})

	t.Run("WriteTo", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			p := make([]byte, 128)
			conn := &quicErrWrapperUDPLikeConn{
				UDPLikeConn: &mocks.UDPLikeConn{
					MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
						return len(p), nil
					},
				},
			}
			count, err := conn.WriteTo(p, &net.UDPAddr{})
			if err != nil {
				t.Fatal(err)
			}
			if count != len(p) {
				t.Fatal("unexpected count")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			p := make([]byte, 128)
			expectedErr := io.EOF
			conn := &quicErrWrapperUDPLikeConn{
				UDPLikeConn: &mocks.UDPLikeConn{
					MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
						return 0, expectedErr
					},
				},
			}
			count, err := conn.WriteTo(p, &net.UDPAddr{})
			if err == nil || err.Error() != FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if count != 0 {
				t.Fatal("unexpected count")
			}
		})
	})

	t.Run("Close", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			conn := &quicErrWrapperUDPLikeConn{
				UDPLikeConn: &mocks.UDPLikeConn{
					MockClose: func() error {
						return nil
					},
				},
			}
			err := conn.Close()
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expectedErr := io.EOF
			conn := &quicErrWrapperUDPLikeConn{
				UDPLikeConn: &mocks.UDPLikeConn{
					MockClose: func() error {
						return expectedErr
					},
				},
			}
			err := conn.Close()
			if err == nil || err.Error() != FailureEOFError {
				t.Fatal("unexpected err", err)
			}
		})
	})
}

func TestQUICDialerErrWrapper(t *testing.T) {
	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		d := &quicDialerErrWrapper{
			QUICDialer: &mocks.QUICDialer{
				MockCloseIdleConnections: func() {
					called = true
				},
			},
		}
		d.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("DialContext", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			expectedConn := &mocks.QUICEarlyConnection{}
			d := &quicDialerErrWrapper{
				QUICDialer: &mocks.QUICDialer{
					MockDialContext: func(ctx context.Context, network, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
						return expectedConn, nil
					},
				},
			}
			ctx := context.Background()
			qconn, err := d.DialContext(ctx, "", "", &tls.Config{}, &quic.Config{})
			if err != nil {
				t.Fatal(err)
			}
			if qconn != expectedConn {
				t.Fatal("unexpected connection")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expectedErr := io.EOF
			d := &quicDialerErrWrapper{
				QUICDialer: &mocks.QUICDialer{
					MockDialContext: func(ctx context.Context, network, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
						return nil, expectedErr
					},
				},
			}
			ctx := context.Background()
			qconn, err := d.DialContext(ctx, "", "", &tls.Config{}, &quic.Config{})
			if err == nil || err.Error() != FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if qconn != nil {
				t.Fatal("unexpected connection")
			}
		})
	})
}
