package mocks

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestUnderlyingNetwork(t *testing.T) {
	t.Run("DefaultCertPool", func(t *testing.T) {
		expect := x509.NewCertPool()
		un := &UnderlyingNetwork{
			MockDefaultCertPool: func() *x509.CertPool {
				return expect
			},
		}
		got := un.DefaultCertPool()
		if got != expect {
			t.Fatal("unexpected result")
		}
	})

	t.Run("DialContext", func(t *testing.T) {
		expect := errors.New("mocked error")
		un := &UnderlyingNetwork{
			MockDialContext: func(ctx context.Context, timeout time.Duration, network, address string) (net.Conn, error) {
				return nil, expect
			},
		}
		ctx := context.Background()
		conn, err := un.DialContext(ctx, time.Second, "tcp", "1.1.1.1:443")
		if !errors.Is(err, expect) {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("ListenUDP", func(t *testing.T) {
		expect := errors.New("mocked error")
		un := &UnderlyingNetwork{
			MockListenUDP: func(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
				return nil, expect
			},
		}
		pconn, err := un.ListenUDP("udp", &net.UDPAddr{})
		if !errors.Is(err, expect) {
			t.Fatal("unexpected err", err)
		}
		if pconn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("GetaddrinfoLookupANY", func(t *testing.T) {
		expect := errors.New("mocked error")
		un := &UnderlyingNetwork{
			MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
				return nil, "", expect
			},
		}
		ctx := context.Background()
		addrs, cname, err := un.GetaddrinfoLookupANY(ctx, "dns.google")
		if !errors.Is(err, expect) {
			t.Fatal("unexpected err", err)
		}
		if len(addrs) != 0 {
			t.Fatal("expected zero length addrs")
		}
		if cname != "" {
			t.Fatal("expected empty name")
		}
	})

	t.Run("GetaddrinfoResolverNetwork", func(t *testing.T) {
		expect := "antani"
		un := &UnderlyingNetwork{
			MockGetaddrinfoResolverNetwork: func() string {
				return expect
			},
		}
		got := un.GetaddrinfoResolverNetwork()
		if got != expect {
			t.Fatal("unexpected resolver network")
		}
	})
}
