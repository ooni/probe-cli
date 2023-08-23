package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/quic-go/quic-go"
)

func TestNetx(t *testing.T) {
	t.Run("NewStdlibResolver", func(t *testing.T) {
		expected := errors.New("mocked error")
		netx := &Netx{&mocks.UnderlyingNetwork{
			MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
				return nil, "", expected
			},
			MockGetaddrinfoResolverNetwork: func() string {
				return "antani"
			},
		}}

		reso := netx.NewStdlibResolver(model.DiscardLogger)

		if reso.Network() != "antani" {
			t.Fatal("unexpected network")
		}

		addrs, err := reso.LookupHost(context.Background(), "dns.google")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err")
		}
		if len(addrs) != 0 {
			t.Fatal("unexpected addrs")
		}
	})

	t.Run("NewDialerWithResolver", func(t *testing.T) {
		netx := &Netx{&mocks.UnderlyingNetwork{
			MockDialContext: func(ctx context.Context, timeout time.Duration, network string, address string) (net.Conn, error) {
				conn := &mocks.Conn{
					MockRemoteAddr: func() net.Addr {
						addr := &mocks.Addr{
							MockString: func() string {
								return address
							},
							MockNetwork: func() string {
								return network
							},
						}
						return addr
					},
				}
				return conn, nil
			},
			MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
				return []string{"8.8.8.8"}, "", nil
			},
			MockGetaddrinfoResolverNetwork: func() string {
				return "antani"
			},
		}}

		reso := netx.NewStdlibResolver(model.DiscardLogger)
		dialer := netx.NewDialerWithResolver(model.DiscardLogger, reso)

		conn, err := dialer.DialContext(context.Background(), "tcp", "dns.google:443")
		if err != nil {
			t.Fatal(err)
		}

		remoteAddr := conn.RemoteAddr()
		if remoteAddr.String() != "8.8.8.8:443" {
			t.Fatal("unexpected remote addr string")
		}
		if remoteAddr.Network() != "tcp" {
			t.Fatal("unexpected remote addr network")
		}
	})

	t.Run("NewQUICListener", func(t *testing.T) {
		expected := errors.New("mocked error")
		netx := &Netx{&mocks.UnderlyingNetwork{
			MockListenUDP: func(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
				return nil, expected
			},
		}}

		listener := netx.NewQUICListener()
		conn, err := listener.Listen(&net.UDPAddr{})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err")
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("NewQUICDialerWithResolver", func(t *testing.T) {
		expected := errors.New("mocked error")
		netx := &Netx{&mocks.UnderlyingNetwork{
			MockListenUDP: func(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
				return nil, expected
			},
			MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
				return []string{"8.8.8.8"}, "", nil
			},
			MockGetaddrinfoResolverNetwork: func() string {
				return "antani"
			},
		}}

		reso := netx.NewStdlibResolver(model.DiscardLogger)
		ql := netx.NewQUICListener()
		dialer := netx.NewQUICDialerWithResolver(ql, model.DiscardLogger, reso)

		conn, err := dialer.DialContext(context.Background(), "dns.google:443", &tls.Config{}, &quic.Config{})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err")
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})
}
