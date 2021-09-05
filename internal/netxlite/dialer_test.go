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
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestDialerResolverNoPort(t *testing.T) {
	dialer := &DialerResolver{Dialer: &net.Dialer{}, Resolver: DefaultResolver}
	conn, err := dialer.DialContext(context.Background(), "tcp", "ooni.nu")
	if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestDialerResolverLookupHostAddress(t *testing.T) {
	dialer := &DialerResolver{Dialer: new(net.Dialer), Resolver: &mocks.Resolver{
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
	dialer := &DialerResolver{Dialer: new(net.Dialer), Resolver: &mocks.Resolver{
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
	dialer := &DialerResolver{Dialer: &mocks.Dialer{
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
	dialer := &DialerResolver{
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
	dialer := &DialerResolver{Dialer: &mocks.Dialer{
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

func TestDialerLoggerSuccess(t *testing.T) {
	d := &DialerLogger{
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
	d := &DialerLogger{
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

func TestDefaultDialerHasTimeout(t *testing.T) {
	expected := 15 * time.Second
	if DefaultDialer.Timeout != expected {
		t.Fatal("unexpected timeout value")
	}
}
