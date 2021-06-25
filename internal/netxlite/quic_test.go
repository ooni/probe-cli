package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"strings"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxmocks"
)

func TestQUICDialerQUICGoCannotSplitHostPort(t *testing.T) {
	tlsConfig := &tls.Config{
		NextProtos: []string{"h3"},
		ServerName: "www.google.com",
	}
	systemdialer := QUICDialerQUICGo{
		QUICListener: &QUICListenerStdlib{},
	}
	ctx := context.Background()
	sess, err := systemdialer.DialContext(
		ctx, "udp", "a.b.c.d", tlsConfig, &quic.Config{})
	if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
		t.Fatal("not the error we expected", err)
	}
	if sess != nil {
		t.Fatal("expected nil sess here")
	}
}

func TestQUICDialerQUICGoInvalidPort(t *testing.T) {
	tlsConfig := &tls.Config{
		NextProtos: []string{"h3"},
		ServerName: "www.google.com",
	}
	systemdialer := QUICDialerQUICGo{
		QUICListener: &QUICListenerStdlib{},
	}
	ctx := context.Background()
	sess, err := systemdialer.DialContext(
		ctx, "udp", "8.8.4.4:xyz", tlsConfig, &quic.Config{})
	if err == nil || !strings.HasSuffix(err.Error(), "invalid syntax") {
		t.Fatal("not the error we expected", err)
	}
	if sess != nil {
		t.Fatal("expected nil sess here")
	}
}

func TestQUICDialerQUICGoInvalidIP(t *testing.T) {
	tlsConfig := &tls.Config{
		NextProtos: []string{"h3"},
		ServerName: "www.google.com",
	}
	systemdialer := QUICDialerQUICGo{
		QUICListener: &QUICListenerStdlib{},
	}
	ctx := context.Background()
	sess, err := systemdialer.DialContext(
		ctx, "udp", "a.b.c.d:0", tlsConfig, &quic.Config{})
	if !errors.Is(err, errInvalidIP) {
		t.Fatal("not the error we expected", err)
	}
	if sess != nil {
		t.Fatal("expected nil sess here")
	}
}

func TestQUICDialerQUICGoCannotListen(t *testing.T) {
	expected := errors.New("mocked error")
	tlsConfig := &tls.Config{
		NextProtos: []string{"h3"},
		ServerName: "www.google.com",
	}
	systemdialer := QUICDialerQUICGo{
		QUICListener: &netxmocks.QUICListener{
			MockListen: func(addr *net.UDPAddr) (net.PacketConn, error) {
				return nil, expected
			},
		},
	}
	ctx := context.Background()
	sess, err := systemdialer.DialContext(
		ctx, "udp", "8.8.8.8:443", tlsConfig, &quic.Config{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if sess != nil {
		t.Fatal("expected nil sess here")
	}
}

func TestQUICDialerWorksAsIntended(t *testing.T) {
	tlsConfig := &tls.Config{
		NextProtos: []string{"h3"},
		ServerName: "dns.google",
	}
	systemdialer := QUICDialerQUICGo{
		QUICListener: &QUICListenerStdlib{},
	}
	ctx := context.Background()
	sess, err := systemdialer.DialContext(
		ctx, "udp", "8.8.8.8:443", tlsConfig, &quic.Config{})
	if err != nil {
		t.Fatal("not the error we expected", err)
	}
	if err := sess.CloseWithError(0, ""); err != nil {
		log.Fatal(err)
	}
}
