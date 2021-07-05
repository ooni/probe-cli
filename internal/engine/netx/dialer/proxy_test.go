package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/netxmocks"
)

func TestProxyDialerDialContextNoProxyURL(t *testing.T) {
	expected := errors.New("mocked error")
	d := &ProxyDialer{
		Dialer: &netxmocks.Dialer{
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
	d := &ProxyDialer{
		ProxyURL: &url.URL{Scheme: "antani"},
	}
	conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, ErrProxyUnsupportedScheme) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("conn is not nil")
	}
}

func TestProxyDialerDialContextWithEOF(t *testing.T) {
	const expect = "10.0.0.1:9050"
	d := &ProxyDialer{
		Dialer: &netxmocks.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				if address != expect {
					return nil, errors.New("unexpected address")
				}
				return nil, io.EOF
			},
		},
		ProxyURL: &url.URL{Scheme: "socks5", Host: expect},
	}
	conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("conn is not nil")
	}
}

func TestProxyDialWrapperPanics(t *testing.T) {
	d := &proxyDialerWrapper{}
	err := func() (rv error) {
		defer func() {
			if r := recover(); r != nil {
				rv = r.(error)
			}
		}()
		d.Dial("tcp", "10.0.0.1:1234")
		return
	}()
	if err.Error() != "proxyDialerWrapper.Dial should not be called directly" {
		t.Fatal("unexpected result", err)
	}
}
