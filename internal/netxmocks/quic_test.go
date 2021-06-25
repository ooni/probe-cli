package netxmocks

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"testing"

	"github.com/lucas-clemente/quic-go"
)

func TestQUICListenerListen(t *testing.T) {
	expected := errors.New("mocked error")
	ql := &QUICListener{
		MockListen: func(addr *net.UDPAddr) (net.PacketConn, error) {
			return nil, expected
		},
	}
	pconn, err := ql.Listen(&net.UDPAddr{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", expected)
	}
	if pconn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestQUICContextDialerDialContext(t *testing.T) {
	expected := errors.New("mocked error")
	qcd := &QUICContextDialer{
		MockDialContext: func(ctx context.Context, network string, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
			return nil, expected
		},
	}
	ctx := context.Background()
	tlsConfig := &tls.Config{}
	quicConfig := &quic.Config{}
	sess, err := qcd.DialContext(ctx, "udp", "dns.google:443", tlsConfig, quicConfig)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if sess != nil {
		t.Fatal("expected nil session")
	}
}
