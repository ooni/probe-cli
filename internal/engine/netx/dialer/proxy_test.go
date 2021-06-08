package dialer_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/mockablex"
)

func TestProxyDialerDialContextNoProxyURL(t *testing.T) {
	expected := errors.New("mocked error")
	d := dialer.ProxyDialer{
		Dialer: mockablex.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, expected
			},
		},
	}
	conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, expected) {
		t.Fatal(err)
	}
	if conn != nil {
		t.Fatal("conn is not nil")
	}
}

func TestProxyDialerDialContextInvalidScheme(t *testing.T) {
	d := dialer.ProxyDialer{
		ProxyURL: &url.URL{Scheme: "antani"},
	}
	conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
	if err.Error() != "Scheme is not socks5" {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("conn is not nil")
	}
}

func TestProxyDialerDialContextWithEOF(t *testing.T) {
	d := dialer.ProxyDialer{
		Dialer: mockablex.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, io.EOF
			},
		},
		ProxyURL: &url.URL{Scheme: "socks5"},
	}
	conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("conn is not nil")
	}
}

func TestProxyDialerDialContextWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately fail
	d := dialer.ProxyDialer{
		Dialer: mockablex.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, io.EOF
			},
		},
		ProxyURL: &url.URL{Scheme: "socks5"},
	}
	conn, err := d.DialContext(ctx, "tcp", "www.google.com:443")
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("conn is not nil")
	}
}

func TestProxyDialerDialContextWithDialerSuccess(t *testing.T) {
	d := dialer.ProxyDialer{
		Dialer: mockablex.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return &mockablex.Conn{
					MockRead: func(b []byte) (int, error) {
						return 0, io.EOF
					},
					MockWrite: func(b []byte) (int, error) {
						return 0, io.EOF
					},
					MockClose: func() error {
						return io.EOF
					},
				}, nil
			},
		},
		ProxyURL: &url.URL{Scheme: "socks5"},
	}
	conn, err := d.DialContextWithDialer(
		context.Background(), dialer.ProxyDialerWrapper{
			Dialer: d.Dialer,
		}, "tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestProxyDialerDialContextWithDialerCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Stop immediately. The FakeDialer sleeps for some microseconds so
	// it is much more likely we immediately exit with done context. The
	// arm where we receive the conn is much less likely.
	cancel()
	d := dialer.ProxyDialer{
		Dialer: mockablex.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				time.Sleep(10 * time.Microsecond)
				return &mockablex.Conn{
					MockRead: func(b []byte) (int, error) {
						return 0, io.EOF
					},
					MockWrite: func(b []byte) (int, error) {
						return 0, io.EOF
					},
					MockClose: func() error {
						return io.EOF
					},
				}, nil
			},
		},
		ProxyURL: &url.URL{Scheme: "socks5"},
	}
	conn, err := d.DialContextWithDialer(
		ctx, dialer.ProxyDialerWrapper{
			Dialer: d.Dialer,
		}, "tcp", "www.google.com:443")
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestProxyDialerWrapper(t *testing.T) {
	d := dialer.ProxyDialerWrapper{
		Dialer: mockablex.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, io.EOF
			},
		},
	}
	conn, err := d.Dial("tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("conn is not nil")
	}
}
