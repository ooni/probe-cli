package filtering

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestTLSProxy(t *testing.T) {
	newproxy := func(action TLSAction) (net.Listener, <-chan interface{}, error) {
		p := &TLSProxy{
			OnIncomingSNI: func(sni string) TLSAction {
				return action
			},
		}
		return p.start("127.0.0.1:0")
	}

	dialTLS := func(ctx context.Context, endpoint string, sni string) (net.Conn, error) {
		d := netxlite.NewDialerWithoutResolver(log.Log)
		th := netxlite.NewTLSHandshakerStdlib(log.Log)
		tdx := netxlite.NewTLSDialerWithConfig(d, th, &tls.Config{
			ServerName: sni,
			NextProtos: []string{"h2", "http/1.1"},
			RootCAs:    netxlite.NewDefaultCertPool(),
		})
		return tdx.DialTLSContext(ctx, "tcp", endpoint)
	}

	t.Run("TLSActionProxy with default proxy", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newproxy(TLSActionProxy)
		if err != nil {
			t.Fatal(err)
		}
		conn, err := dialTLS(ctx, listener.Addr().String(), "dns.google")
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("TLSActionTimeout", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newproxy(TLSActionTimeout)
		if err != nil {
			t.Fatal(err)
		}
		conn, err := dialTLS(ctx, listener.Addr().String(), "dns.google")
		if err == nil || err.Error() != netxlite.FailureGenericTimeoutError {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("TLSActionAlertInternalError", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newproxy(TLSActionAlertInternalError)
		if err != nil {
			t.Fatal(err)
		}
		conn, err := dialTLS(ctx, listener.Addr().String(), "dns.google")
		if err == nil || !strings.HasSuffix(err.Error(), "tls: internal error") {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("TLSActionAlertUnrecognizedName", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newproxy(TLSActionAlertUnrecognizedName)
		if err != nil {
			t.Fatal(err)
		}
		conn, err := dialTLS(ctx, listener.Addr().String(), "dns.google")
		if err == nil || !strings.HasSuffix(err.Error(), "tls: unrecognized name") {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("TLSActionEOF", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newproxy(TLSActionEOF)
		if err != nil {
			t.Fatal(err)
		}
		conn, err := dialTLS(ctx, listener.Addr().String(), "dns.google")
		if err == nil || err.Error() != netxlite.FailureEOFError {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("TLSActionReset", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newproxy(TLSActionReset)
		if err != nil {
			t.Fatal(err)
		}
		conn, err := dialTLS(ctx, listener.Addr().String(), "dns.google")
		if err == nil || err.Error() != netxlite.FailureConnectionReset {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("handle cannot read ClientHello", func(t *testing.T) {
		listener, done, err := newproxy(TLSActionProxy)
		if err != nil {
			t.Fatal(err)
		}
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		conn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
		buff := make([]byte, 1<<17)
		_, err = conn.Read(buff)
		if err == nil || !strings.HasSuffix(err.Error(), "connection reset by peer") {
			t.Fatal("unexpected err", err)
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("TLSActionProxy fails because we don't have SNI", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newproxy(TLSActionProxy)
		if err != nil {
			t.Fatal(err)
		}
		conn, err := dialTLS(ctx, listener.Addr().String(), "127.0.0.1")
		if err == nil || err.Error() != netxlite.FailureConnectionReset {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("TLSActionProxy fails because we can't dial", func(t *testing.T) {
		ctx := context.Background()
		listener, done, err := newproxy(TLSActionProxy)
		if err != nil {
			t.Fatal(err)
		}
		conn, err := dialTLS(ctx, listener.Addr().String(), "antani.ooni.org")
		if err == nil || err.Error() != netxlite.FailureConnectionReset {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		listener.Close()
		<-done // wait for background goroutine to exit
	})

	t.Run("proxydial fails because it's connecting to itself", func(t *testing.T) {
		p := &TLSProxy{}
		conn := &mocks.Conn{
			MockClose: func() error {
				return nil
			},
		}
		p.proxydial(conn, "ooni.org", nil, func(network, address string) (net.Conn, error) {
			return &mocks.Conn{
				MockClose: func() error {
					return nil
				},
				MockLocalAddr: func() net.Addr {
					return &net.TCPAddr{
						IP: net.IPv6loopback,
					}
				},
				MockRemoteAddr: func() net.Addr {
					return &net.TCPAddr{
						IP: net.IPv6loopback,
					}
				},
			}, nil
		})
	})

	t.Run("proxydial fails because it cannot write the hello", func(t *testing.T) {
		p := &TLSProxy{}
		conn := &mocks.Conn{
			MockClose: func() error {
				return nil
			},
		}
		p.proxydial(conn, "ooni.org", nil, func(network, address string) (net.Conn, error) {
			return &mocks.Conn{
				MockClose: func() error {
					return nil
				},
				MockLocalAddr: func() net.Addr {
					return &net.TCPAddr{
						IP: net.IPv6loopback,
					}
				},
				MockRemoteAddr: func() net.Addr {
					return &net.TCPAddr{
						IP: net.IPv4(10, 0, 0, 1),
					}
				},
				MockWrite: func(b []byte) (int, error) {
					return 0, errors.New("mocked error")
				},
			}, nil
		})
	})

	t.Run("Start fails on an invalid address", func(t *testing.T) {
		p := &TLSProxy{}
		listener, err := p.Start("127.0.0.1")
		if err == nil {
			t.Fatal("expected an error")
		}
		if listener != nil {
			t.Fatal("expected nil listener")
		}
	})
}
