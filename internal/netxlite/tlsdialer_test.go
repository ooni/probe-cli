package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestTLSDialerFailureSplitHostPort(t *testing.T) {
	dialer := &TLSDialer{}
	ctx := context.Background()
	const address = "www.google.com" // missing port
	conn, err := dialer.DialTLSContext(ctx, "tcp", address)
	if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func TestTLSDialerFailureDialing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately fail
	dialer := TLSDialer{Dialer: &net.Dialer{}}
	conn, err := dialer.DialTLSContext(ctx, "tcp", "www.google.com:443")
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func TestTLSDialerFailureHandshaking(t *testing.T) {
	ctx := context.Background()
	dialer := TLSDialer{
		Config: &tls.Config{},
		Dialer: &mocks.Dialer{MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return &mocks.Conn{MockWrite: func(b []byte) (int, error) {
				return 0, io.EOF
			}, MockClose: func() error {
				return nil
			}, MockSetDeadline: func(t time.Time) error {
				return nil
			}}, nil
		}},
		TLSHandshaker: &TLSHandshakerConfigurable{},
	}
	conn, err := dialer.DialTLSContext(ctx, "tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func TestTLSDialerSuccessHandshaking(t *testing.T) {
	ctx := context.Background()
	dialer := TLSDialer{
		Dialer: &mocks.Dialer{MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return &mocks.Conn{MockWrite: func(b []byte) (int, error) {
				return 0, io.EOF
			}, MockClose: func() error {
				return nil
			}, MockSetDeadline: func(t time.Time) error {
				return nil
			}}, nil
		}},
		TLSHandshaker: &mocks.TLSHandshaker{
			MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
				return tls.Client(conn, config), tls.ConnectionState{}, nil
			},
		},
	}
	conn, err := dialer.DialTLSContext(ctx, "tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("connection is nil")
	}
	conn.Close()
}

func TestTLSDialerConfigFromEmptyConfigForWeb(t *testing.T) {
	d := &TLSDialer{}
	config := d.config("www.google.com", "443")
	if config.ServerName != "www.google.com" {
		t.Fatal("invalid server name")
	}
	if diff := cmp.Diff(config.NextProtos, []string{"h2", "http/1.1"}); diff != "" {
		t.Fatal(diff)
	}
}

func TestTLSDialerConfigFromEmptyConfigForDoT(t *testing.T) {
	d := &TLSDialer{}
	config := d.config("dns.google", "853")
	if config.ServerName != "dns.google" {
		t.Fatal("invalid server name")
	}
	if diff := cmp.Diff(config.NextProtos, []string{"dot"}); diff != "" {
		t.Fatal(diff)
	}
}

func TestTLSDialerConfigWithServerName(t *testing.T) {
	d := &TLSDialer{
		Config: &tls.Config{
			ServerName: "example.com",
		},
	}
	config := d.config("dns.google", "853")
	if config.ServerName != "example.com" {
		t.Fatal("invalid server name")
	}
	if diff := cmp.Diff(config.NextProtos, []string{"dot"}); diff != "" {
		t.Fatal(diff)
	}
}

func TestTLSDialerConfigWithALPN(t *testing.T) {
	d := &TLSDialer{
		Config: &tls.Config{
			NextProtos: []string{"h2"},
		},
	}
	config := d.config("dns.google", "853")
	if config.ServerName != "dns.google" {
		t.Fatal("invalid server name")
	}
	if diff := cmp.Diff(config.NextProtos, []string{"h2"}); diff != "" {
		t.Fatal(diff)
	}
}
