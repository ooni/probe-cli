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

func TestQUICDialerCannotPerformHandshake(t *testing.T) {
	tlsConfig := &tls.Config{
		NextProtos: []string{"h3"},
		ServerName: "dns.google",
	}
	systemdialer := QUICDialerQUICGo{
		QUICListener: &QUICListenerStdlib{},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	sess, err := systemdialer.DialContext(
		ctx, "udp", "8.8.8.8:443", tlsConfig, &quic.Config{})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if sess != nil {
		log.Fatal("expected nil session here")
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
	<-sess.HandshakeComplete().Done()
	if err := sess.CloseWithError(0, ""); err != nil {
		log.Fatal(err)
	}
}

func TestQUICDialerResolverSuccess(t *testing.T) {
	tlsConfig := &tls.Config{NextProtos: []string{"h3"}}
	dialer := &QUICDialerResolver{
		Resolver: &net.Resolver{}, Dialer: &QUICDialerQUICGo{
			QUICListener: &QUICListenerStdlib{},
		}}
	sess, err := dialer.DialContext(
		context.Background(), "udp", "www.google.com:443",
		tlsConfig, &quic.Config{})
	if err != nil {
		t.Fatal(err)
	}
	<-sess.HandshakeComplete().Done()
	if err := sess.CloseWithError(0, ""); err != nil {
		t.Fatal(err)
	}
}

func TestQUICDialerResolverNoPort(t *testing.T) {
	tlsConfig := &tls.Config{NextProtos: []string{"h3"}}
	dialer := &QUICDialerResolver{
		Resolver: new(net.Resolver), Dialer: &QUICDialerQUICGo{}}
	sess, err := dialer.DialContext(
		context.Background(), "udp", "www.google.com",
		tlsConfig, &quic.Config{})
	if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
		t.Fatal("not the error we expected")
	}
	if sess != nil {
		t.Fatal("expected a nil sess here")
	}
}

func TestQUICDialerResolverLookupHostAddress(t *testing.T) {
	dialer := &QUICDialerResolver{Resolver: &netxmocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			// We should not arrive here and call this function but if we do then
			// there is going to be an error that fails this test.
			return nil, errors.New("mocked error")
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

func TestQUICDialerResolverLookupHostFailure(t *testing.T) {
	tlsConfig := &tls.Config{NextProtos: []string{"h3"}}
	expected := errors.New("mocked error")
	dialer := &QUICDialerResolver{Resolver: &netxmocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return nil, expected
		},
	}}
	sess, err := dialer.DialContext(
		context.Background(), "udp", "dns.google.com:853",
		tlsConfig, &quic.Config{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if sess != nil {
		t.Fatal("expected nil sess")
	}
}

func TestQUICDialerResolverInvalidPort(t *testing.T) {
	// This test allows us to check for the case where every attempt
	// to establish a connection leads to a failure
	tlsConf := &tls.Config{NextProtos: []string{"h3"}}
	dialer := &QUICDialerResolver{
		Resolver: new(net.Resolver), Dialer: &QUICDialerQUICGo{
			QUICListener: &QUICListenerStdlib{},
		}}
	sess, err := dialer.DialContext(
		context.Background(), "udp", "www.google.com:0",
		tlsConf, &quic.Config{})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if !strings.HasSuffix(err.Error(), "sendto: invalid argument") &&
		!strings.HasSuffix(err.Error(), "sendto: can't assign requested address") {
		t.Fatal("not the error we expected", err)
	}
	if sess != nil {
		t.Fatal("expected nil sess")
	}
}
