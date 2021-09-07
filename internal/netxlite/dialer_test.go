package netxlite

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestDialerSystemCloseIdleConnections(t *testing.T) {
	d := &dialerSystem{}
	d.CloseIdleConnections() // should not crash
}

func TestDialerResolverNoPort(t *testing.T) {
	dialer := &dialerResolver{Dialer: defaultDialer, Resolver: DefaultResolver}
	conn, err := dialer.DialContext(context.Background(), "tcp", "ooni.nu")
	if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestDialerResolverLookupHostAddress(t *testing.T) {
	dialer := &dialerResolver{Dialer: defaultDialer, Resolver: &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return nil, errors.New("we should not call this function")
		},
	}}
	addrs, err := dialer.lookupHost(context.Background(), "1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "1.1.1.1" {
		t.Fatal("not the result we expected")
	}
}

func TestDialerResolverLookupHostFailure(t *testing.T) {
	expected := errors.New("mocked error")
	dialer := &dialerResolver{Dialer: defaultDialer, Resolver: &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return nil, expected
		},
	}}
	ctx := context.Background()
	conn, err := dialer.DialContext(ctx, "tcp", "dns.google.com:853")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn")
	}
}

func TestDialerResolverDialForSingleIPFails(t *testing.T) {
	dialer := &dialerResolver{Dialer: &mocks.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return nil, io.EOF
		},
	}, Resolver: DefaultResolver}
	conn, err := dialer.DialContext(context.Background(), "tcp", "1.1.1.1:853")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn")
	}
}

func TestDialerResolverDialForManyIPFails(t *testing.T) {
	dialer := &dialerResolver{
		Dialer: &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, io.EOF
			},
		}, Resolver: &mocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return []string{"1.1.1.1", "8.8.8.8"}, nil
			},
		}}
	conn, err := dialer.DialContext(context.Background(), "tcp", "dot.dns:853")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn")
	}
}

func TestDialerResolverDialForManyIPSuccess(t *testing.T) {
	dialer := &dialerResolver{Dialer: &mocks.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return &mocks.Conn{
				MockClose: func() error {
					return nil
				},
			}, nil
		},
	}, Resolver: &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return []string{"1.1.1.1", "8.8.8.8"}, nil
		},
	}}
	conn, err := dialer.DialContext(context.Background(), "tcp", "dot.dns:853")
	if err != nil {
		t.Fatal("expected nil error here")
	}
	if conn == nil {
		t.Fatal("expected non-nil conn")
	}
	conn.Close()
}

func TestDialerResolverCloseIdleConnections(t *testing.T) {
	var (
		calledDialer   bool
		calledResolver bool
	)
	d := &dialerResolver{
		Dialer: &mocks.Dialer{
			MockCloseIdleConnections: func() {
				calledDialer = true
			},
		},
		Resolver: &mocks.Resolver{
			MockCloseIdleConnections: func() {
				calledResolver = true
			},
		},
	}
	d.CloseIdleConnections()
	if !calledDialer || !calledResolver {
		t.Fatal("not called")
	}
}

func TestDialerLoggerSuccess(t *testing.T) {
	d := &dialerLogger{
		Dialer: &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return &mocks.Conn{
					MockClose: func() error {
						return nil
					},
				}, nil
			},
		},
		Logger: log.Log,
	}
	conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn here")
	}
	conn.Close()
}

func TestDialerLoggerFailure(t *testing.T) {
	d := &dialerLogger{
		Dialer: &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, io.EOF
			},
		},
		Logger: log.Log,
	}
	conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestDialerLoggerCloseIdleConnections(t *testing.T) {
	var (
		calledDialer bool
	)
	d := &dialerLogger{
		Dialer: &mocks.Dialer{
			MockCloseIdleConnections: func() {
				calledDialer = true
			},
		},
	}
	d.CloseIdleConnections()
	if !calledDialer {
		t.Fatal("not called")
	}
}

func TestUnderlyingDialerHasTimeout(t *testing.T) {
	expected := 15 * time.Second
	if underlyingDialer.Timeout != expected {
		t.Fatal("unexpected timeout value")
	}
}

func TestNewDialerWithoutResolverChain(t *testing.T) {
	dlr := NewDialerWithoutResolver(log.Log)
	dlog, okay := dlr.(*dialerLogger)
	if !okay {
		t.Fatal("invalid type")
	}
	if dlog.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	dreso, okay := dlog.Dialer.(*dialerResolver)
	if !okay {
		t.Fatal("invalid type")
	}
	if _, okay := dreso.Resolver.(*nullResolver); !okay {
		t.Fatal("invalid Resolver type")
	}
	dlog, okay = dreso.Dialer.(*dialerLogger)
	if !okay {
		t.Fatal("invalid type")
	}
	if dlog.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	dew, okay := dlog.Dialer.(*dialerErrWrapper)
	if !okay {
		t.Fatal("invalid type")
	}
	if _, okay := dew.Dialer.(*dialerSystem); !okay {
		t.Fatal("invalid type")
	}
}

func TestNewSingleUseDialerWorksAsIntended(t *testing.T) {
	conn := &mocks.Conn{}
	d := NewSingleUseDialer(conn)
	outconn, err := d.DialContext(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if conn != outconn {
		t.Fatal("invalid outconn")
	}
	for i := 0; i < 4; i++ {
		outconn, err = d.DialContext(context.Background(), "", "")
		if !errors.Is(err, ErrNoConnReuse) {
			t.Fatal("not the error we expected", err)
		}
		if outconn != nil {
			t.Fatal("expected nil outconn here")
		}
	}
}

func TestDialerErrWrapper(t *testing.T) {
	t.Run("DialContext on success", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			expectedConn := &mocks.Conn{}
			d := &dialerErrWrapper{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
						return expectedConn, nil
					},
				},
			}
			ctx := context.Background()
			conn, err := d.DialContext(ctx, "", "")
			if err != nil {
				t.Fatal(err)
			}
			errWrapperConn := conn.(*dialerErrWrapperConn)
			if errWrapperConn.Conn != expectedConn {
				t.Fatal("unexpected conn")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expectedErr := io.EOF
			d := &dialerErrWrapper{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
						return nil, expectedErr
					},
				},
			}
			ctx := context.Background()
			conn, err := d.DialContext(ctx, "", "")
			if err == nil || err.Error() != errorsx.FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		d := &dialerErrWrapper{
			Dialer: &mocks.Dialer{
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
}

func TestDialerErrWrapperConn(t *testing.T) {
	t.Run("Read", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			b := make([]byte, 128)
			conn := &dialerErrWrapperConn{
				Conn: &mocks.Conn{
					MockRead: func(b []byte) (int, error) {
						return len(b), nil
					},
				},
			}
			count, err := conn.Read(b)
			if err != nil {
				t.Fatal(err)
			}
			if count != len(b) {
				t.Fatal("unexpected count")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			b := make([]byte, 128)
			expectedErr := io.EOF
			conn := &dialerErrWrapperConn{
				Conn: &mocks.Conn{
					MockRead: func(b []byte) (int, error) {
						return 0, expectedErr
					},
				},
			}
			count, err := conn.Read(b)
			if err == nil || err.Error() != errorsx.FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if count != 0 {
				t.Fatal("unexpected count")
			}
		})
	})

	t.Run("Write", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			b := make([]byte, 128)
			conn := &dialerErrWrapperConn{
				Conn: &mocks.Conn{
					MockWrite: func(b []byte) (int, error) {
						return len(b), nil
					},
				},
			}
			count, err := conn.Write(b)
			if err != nil {
				t.Fatal(err)
			}
			if count != len(b) {
				t.Fatal("unexpected count")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			b := make([]byte, 128)
			expectedErr := io.EOF
			conn := &dialerErrWrapperConn{
				Conn: &mocks.Conn{
					MockWrite: func(b []byte) (int, error) {
						return 0, expectedErr
					},
				},
			}
			count, err := conn.Write(b)
			if err == nil || err.Error() != errorsx.FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if count != 0 {
				t.Fatal("unexpected count")
			}
		})
	})

	t.Run("Close", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			conn := &dialerErrWrapperConn{
				Conn: &mocks.Conn{
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
			conn := &dialerErrWrapperConn{
				Conn: &mocks.Conn{
					MockClose: func() error {
						return expectedErr
					},
				},
			}
			err := conn.Close()
			if err == nil || err.Error() != errorsx.FailureEOFError {
				t.Fatal("unexpected err", err)
			}
		})
	})
}
