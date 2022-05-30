package netxlite

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestMaybeProxyDialer(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		t.Run("missing proxy URL", func(t *testing.T) {
			expected := errors.New("mocked error")
			d := &MaybeProxyDialer{
				Dialer: &mocks.Dialer{MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
					return nil, expected
				}},
				ProxyURL: nil,
			}
			conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
			if !errors.Is(err, expected) {
				t.Fatal(err)
			}
			if conn != nil {
				t.Fatal("conn is not nil")
			}
		})

		t.Run("invalid scheme", func(t *testing.T) {
			child := &mocks.Dialer{}
			URL := &url.URL{Scheme: "antani"}
			d := NewMaybeProxyDialer(child, URL)
			conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
			if !errors.Is(err, ErrProxyUnsupportedScheme) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("conn is not nil")
			}
		})

		t.Run("underlying dial fails with EOF", func(t *testing.T) {
			const expect = "10.0.0.1:9050"
			d := &MaybeProxyDialer{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
						if address != expect {
							return nil, errors.New("unexpected address")
						}
						return nil, io.EOF
					},
				},
				ProxyURL: &url.URL{
					Scheme: "socks5",
					Host:   expect,
				},
			}
			conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
			if !errors.Is(err, io.EOF) {
				t.Fatal("not the error we expected")
			}
			if conn != nil {
				t.Fatal("conn is not nil")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.Dialer{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		URL := &url.URL{}
		dialer := NewMaybeProxyDialer(child, URL)
		dialer.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("proxyDialerWrapper", func(t *testing.T) {
		t.Run("Dial panics", func(t *testing.T) {
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
		})
	})
}
