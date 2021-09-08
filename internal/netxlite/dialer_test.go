package netxlite

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestNewDialer(t *testing.T) {
	t.Run("produces a chain with the expected types", func(t *testing.T) {
		d := NewDialerWithoutResolver(log.Log)
		logger := d.(*dialerLogger)
		if logger.Logger != log.Log {
			t.Fatal("invalid logger")
		}
		reso := logger.Dialer.(*dialerResolver)
		if _, okay := reso.Resolver.(*nullResolver); !okay {
			t.Fatal("invalid Resolver type")
		}
		logger = reso.Dialer.(*dialerLogger)
		if logger.Logger != log.Log {
			t.Fatal("invalid logger")
		}
		errWrapper := logger.Dialer.(*dialerErrWrapper)
		_ = errWrapper.Dialer.(*dialerSystem)
	})
}

func TestDialerSystem(t *testing.T) {
	t.Run("has a default timeout", func(t *testing.T) {
		d := &dialerSystem{}
		ud := d.newUnderlyingDialer()
		if ud.Timeout != dialerDefaultTimeout {
			t.Fatal("unexpected default timeout")
		}
	})

	t.Run("we can change the timeout for testing", func(t *testing.T) {
		const smaller = 1 * time.Second
		d := &dialerSystem{timeout: smaller}
		ud := d.newUnderlyingDialer()
		if ud.Timeout != smaller {
			t.Fatal("unexpected timeout")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		d := &dialerSystem{}
		d.CloseIdleConnections() // to avoid missing coverage
	})

	t.Run("DialContext", func(t *testing.T) {
		t.Run("with canceled context", func(t *testing.T) {
			d := &dialerSystem{}
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // immediately!
			conn, err := d.DialContext(ctx, "tcp", "dns.google:443")
			if err == nil || err.Error() != "dial tcp: operation was canceled" {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("unexpected conn")
			}
		})

		t.Run("enforces the configured timeout", func(t *testing.T) {
			const timeout = 1 * time.Millisecond
			d := &dialerSystem{timeout: timeout}
			ctx := context.Background()
			start := time.Now()
			conn, err := d.DialContext(ctx, "tcp", "dns.google:443")
			stop := time.Now()
			if err == nil || !strings.HasSuffix(err.Error(), "i/o timeout") {
				t.Fatal(err)
			}
			if conn != nil {
				t.Fatal("unexpected conn")
			}
			if stop.Sub(start) > 100*time.Millisecond {
				t.Fatal("undable to enforce timeout")
			}
		})
	})
}

func TestDialerResolver(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		t.Run("fails without a port", func(t *testing.T) {
			d := &dialerResolver{
				Dialer:   &dialerSystem{},
				Resolver: &resolverSystem{},
			}
			const missingPort = "ooni.nu"
			conn, err := d.DialContext(context.Background(), "tcp", missingPort)
			if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("unexpected conn")
			}
		})

		t.Run("handles dialing error correctly for single IP address", func(t *testing.T) {
			d := &dialerResolver{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
						return nil, io.EOF
					},
				},
				Resolver: &nullResolver{},
			}
			conn, err := d.DialContext(context.Background(), "tcp", "1.1.1.1:853")
			if !errors.Is(err, io.EOF) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})

		t.Run("handles dialing error correctly for many IP addresses", func(t *testing.T) {
			d := &dialerResolver{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
						return nil, io.EOF
					},
				},
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"1.1.1.1", "8.8.8.8"}, nil
					},
				},
			}
			conn, err := d.DialContext(context.Background(), "tcp", "dot.dns:853")
			if !errors.Is(err, io.EOF) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})

		t.Run("handles dialing success correctly for many IP addresses", func(t *testing.T) {
			expectedConn := &mocks.Conn{
				MockClose: func() error {
					return nil
				},
			}
			d := &dialerResolver{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
						return expectedConn, nil
					},
				},
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"1.1.1.1", "8.8.8.8"}, nil
					},
				},
			}
			conn, err := d.DialContext(context.Background(), "tcp", "dot.dns:853")
			if err != nil {
				t.Fatal(err)
			}
			if conn != expectedConn {
				t.Fatal("unexpected conn")
			}
			conn.Close()
		})

		t.Run("calls the underlying dialer sequentially", func(t *testing.T) {
			// This test is fundamental to the following
			// TODO(https://github.com/ooni/probe/issues/1779)
			mu := &sync.Mutex{}
			d := &dialerResolver{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
						// It should not happen to have parallel dials with
						// this implementation. When we have parallelism greater
						// than one, this code will lock forever and we'll see
						// a failed test and see we broke the QUIRK.
						defer mu.Unlock()
						mu.Lock()
						return nil, io.EOF
					},
				},
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"1.1.1.1", "8.8.8.8"}, nil
					},
				},
			}
			conn, err := d.DialContext(context.Background(), "tcp", "dot.dns:853")
			if !errors.Is(err, io.EOF) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})

		t.Run("attempts with IPv4 addresses before IPv6 addresses", func(t *testing.T) {
			// This test is fundamental to the following
			// TODO(https://github.com/ooni/probe/issues/1779)
			mu := &sync.Mutex{}
			var attempts []string
			d := &dialerResolver{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
						// It should not happen to have parallel dials with
						// this implementation. When we have parallelism greater
						// than one, this code will lock forever and we'll see
						// a failed test and see we broke the QUIRK.
						defer mu.Unlock()
						attempts = append(attempts, address)
						mu.Lock()
						return nil, io.EOF
					},
				},
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"2001:4860:4860::8888", "8.8.8.8"}, nil
					},
				},
			}
			conn, err := d.DialContext(context.Background(), "tcp", "dot.dns:853")
			if !errors.Is(err, io.EOF) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
			mu.Lock()
			asExpected := (attempts[0] == "8.8.8.8:853" &&
				attempts[1] == "[2001:4860:4860::8888]:853")
			mu.Unlock()
			if !asExpected {
				t.Fatal("addresses not reordered")
			}
		})

		t.Run("returns the first meaningful error if there is one", func(t *testing.T) {
			// This test is fundamental to the following
			// TODO(https://github.com/ooni/probe/issues/1779)
			mu := &sync.Mutex{}
			errorsList := []error{
				errors.New("a mocked error"),
				errorsx.NewErrWrapper(
					errorsx.ClassifyGenericError,
					errorsx.CloseOperation,
					io.EOF,
				),
			}
			var errorIdx int
			d := &dialerResolver{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
						// It should not happen to have parallel dials with
						// this implementation. When we have parallelism greater
						// than one, this code will lock forever and we'll see
						// a failed test and see we broke the QUIRK.
						defer mu.Unlock()
						err := errorsList[errorIdx]
						errorIdx++
						mu.Lock()
						return nil, err
					},
				},
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"2001:4860:4860::8888", "8.8.8.8"}, nil
					},
				},
			}
			conn, err := d.DialContext(context.Background(), "tcp", "dot.dns:853")
			if err == nil || err.Error() != errorsx.FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})

		t.Run("though ignores the unknown failures", func(t *testing.T) {
			// This test is fundamental to the following
			// TODO(https://github.com/ooni/probe/issues/1779)
			mu := &sync.Mutex{}
			errorsList := []error{
				errors.New("a mocked error"),
				errorsx.NewErrWrapper(
					errorsx.ClassifyGenericError,
					errorsx.CloseOperation,
					errors.New("antani"),
				),
			}
			var errorIdx int
			d := &dialerResolver{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
						// It should not happen to have parallel dials with
						// this implementation. When we have parallelism greater
						// than one, this code will lock forever and we'll see
						// a failed test and see we broke the QUIRK.
						defer mu.Unlock()
						err := errorsList[errorIdx]
						errorIdx++
						mu.Lock()
						return nil, err
					},
				},
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"2001:4860:4860::8888", "8.8.8.8"}, nil
					},
				},
			}
			conn, err := d.DialContext(context.Background(), "tcp", "dot.dns:853")
			if err == nil || err.Error() != "a mocked error" {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})
	})

	t.Run("lookupHost", func(t *testing.T) {
		t.Run("handles addresses correctly", func(t *testing.T) {
			dialer := &dialerResolver{
				Dialer:   &dialerSystem{},
				Resolver: &nullResolver{},
			}
			addrs, err := dialer.lookupHost(context.Background(), "1.1.1.1")
			if err != nil {
				t.Fatal(err)
			}
			if len(addrs) != 1 || addrs[0] != "1.1.1.1" {
				t.Fatal("not the result we expected")
			}
		})

		t.Run("fails correctly on lookup error", func(t *testing.T) {
			dialer := &dialerResolver{
				Dialer:   &dialerSystem{},
				Resolver: &nullResolver{},
			}
			ctx := context.Background()
			conn, err := dialer.DialContext(ctx, "tcp", "dns.google.com:853")
			if !errors.Is(err, ErrNoResolver) {
				t.Fatal("not the error we expected", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
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
	})
}

func TestDialerLogger(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		t.Run("handles success correctly", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
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
				Logger: lo,
			}
			conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
			if err != nil {
				t.Fatal(err)
			}
			if conn == nil {
				t.Fatal("expected non-nil conn here")
			}
			conn.Close()
			if count != 2 {
				t.Fatal("not enough log calls")
			}
		})

		t.Run("handles failure correctly", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			d := &dialerLogger{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
						return nil, io.EOF
					},
				},
				Logger: lo,
			}
			conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
			if !errors.Is(err, io.EOF) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("expected nil conn here")
			}
			if count != 2 {
				t.Fatal("not enough log calls")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
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
	})
}

func TestDialerSingleUse(t *testing.T) {
	t.Run("works as intended", func(t *testing.T) {
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
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		d := &dialerSingleUse{}
		d.CloseIdleConnections() // to have the coverage
	})
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

func TestNewNullDialer(t *testing.T) {
	dialer := NewNullDialer()
	conn, err := dialer.DialContext(context.Background(), "", "")
	if !errors.Is(err, ErrNoDialer) {
		t.Fatal("unexpected err", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn")
	}
	dialer.CloseIdleConnections() // to have coverage
}
