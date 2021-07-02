package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxmocks"
	"github.com/ooni/probe-cli/v3/internal/quicx"
)

func TestQUICDialerQUICGoCannotSplitHostPort(t *testing.T) {
	tlsConfig := &tls.Config{
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
		ServerName: "www.google.com",
	}
	systemdialer := QUICDialerQUICGo{
		QUICListener: &netxmocks.QUICListener{
			MockListen: func(addr *net.UDPAddr) (quicx.UDPConn, error) {
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

func TestQUICDialerQUICGoCannotPerformHandshake(t *testing.T) {
	tlsConfig := &tls.Config{
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

func TestQUICDialerQUICGoWorksAsIntended(t *testing.T) {
	tlsConfig := &tls.Config{
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
		t.Fatal(err)
	}
}

func TestQUICDialerQUICGoTLSDefaultsForWeb(t *testing.T) {
	expected := errors.New("mocked error")
	var gotTLSConfig *tls.Config
	tlsConfig := &tls.Config{
		ServerName: "dns.google",
	}
	systemdialer := QUICDialerQUICGo{
		QUICListener: &QUICListenerStdlib{},
		mockDialEarlyContext: func(ctx context.Context, pconn net.PacketConn,
			remoteAddr net.Addr, host string, tlsConfig *tls.Config,
			quicConfig *quic.Config) (quic.EarlySession, error) {
			gotTLSConfig = tlsConfig
			return nil, expected
		},
	}
	ctx := context.Background()
	sess, err := systemdialer.DialContext(
		ctx, "udp", "8.8.8.8:443", tlsConfig, &quic.Config{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if sess != nil {
		t.Fatal("expected nil session here")
	}
	if tlsConfig.RootCAs != nil {
		t.Fatal("tlsConfig.RootCAs should not have been changed")
	}
	if gotTLSConfig.RootCAs != defaultCertPool {
		t.Fatal("invalid gotTLSConfig.RootCAs")
	}
	if tlsConfig.NextProtos != nil {
		t.Fatal("tlsConfig.NextProtos should not have been changed")
	}
	if diff := cmp.Diff(gotTLSConfig.NextProtos, []string{"h3"}); diff != "" {
		t.Fatal("invalid gotTLSConfig.NextProtos", diff)
	}
	if tlsConfig.ServerName != gotTLSConfig.ServerName {
		t.Fatal("the ServerName field must match")
	}
}

func TestQUICDialerQUICGoTLSDefaultsForDoQ(t *testing.T) {
	expected := errors.New("mocked error")
	var gotTLSConfig *tls.Config
	tlsConfig := &tls.Config{
		ServerName: "dns.google",
	}
	systemdialer := QUICDialerQUICGo{
		QUICListener: &QUICListenerStdlib{},
		mockDialEarlyContext: func(ctx context.Context, pconn net.PacketConn,
			remoteAddr net.Addr, host string, tlsConfig *tls.Config,
			quicConfig *quic.Config) (quic.EarlySession, error) {
			gotTLSConfig = tlsConfig
			return nil, expected
		},
	}
	ctx := context.Background()
	sess, err := systemdialer.DialContext(
		ctx, "udp", "8.8.8.8:8853", tlsConfig, &quic.Config{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if sess != nil {
		t.Fatal("expected nil session here")
	}
	if tlsConfig.RootCAs != nil {
		t.Fatal("tlsConfig.RootCAs should not have been changed")
	}
	if gotTLSConfig.RootCAs != defaultCertPool {
		t.Fatal("invalid gotTLSConfig.RootCAs")
	}
	if tlsConfig.NextProtos != nil {
		t.Fatal("tlsConfig.NextProtos should not have been changed")
	}
	if diff := cmp.Diff(gotTLSConfig.NextProtos, []string{"dq"}); diff != "" {
		t.Fatal("invalid gotTLSConfig.NextProtos", diff)
	}
	if tlsConfig.ServerName != gotTLSConfig.ServerName {
		t.Fatal("the ServerName field must match")
	}
}

func TestQUICDialerResolverSuccess(t *testing.T) {
	tlsConfig := &tls.Config{}
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
	tlsConfig := &tls.Config{}
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
	tlsConfig := &tls.Config{}
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
	tlsConf := &tls.Config{}
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

func TestQUICDialerResolverApplyTLSDefaults(t *testing.T) {
	expected := errors.New("mocked error")
	var gotTLSConfig *tls.Config
	tlsConfig := &tls.Config{}
	dialer := &QUICDialerResolver{
		Resolver: new(net.Resolver), Dialer: &netxmocks.QUICContextDialer{
			MockDialContext: func(ctx context.Context, network, address string,
				tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
				gotTLSConfig = tlsConfig
				return nil, expected
			},
		}}
	sess, err := dialer.DialContext(
		context.Background(), "udp", "www.google.com:443",
		tlsConfig, &quic.Config{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if sess != nil {
		t.Fatal("expected nil session here")
	}
	if tlsConfig.ServerName != "" {
		t.Fatal("should not have changed tlsConfig.ServerName")
	}
	if gotTLSConfig.ServerName != "www.google.com" {
		t.Fatal("gotTLSConfig.ServerName has not been set")
	}
}

func TestQUICDialerLoggerSuccess(t *testing.T) {
	d := &QUICDialerLogger{
		Dialer: &netxmocks.QUICContextDialer{
			MockDialContext: func(ctx context.Context, network string,
				address string, tlsConfig *tls.Config,
				quicConfig *quic.Config) (quic.EarlySession, error) {
				return &netxmocks.QUICEarlySession{
					MockCloseWithError: func(
						code quic.ApplicationErrorCode, reason string) error {
						return nil
					},
				}, nil
			},
		},
		Logger: log.Log,
	}
	ctx := context.Background()
	tlsConfig := &tls.Config{}
	quicConfig := &quic.Config{}
	sess, err := d.DialContext(ctx, "udp", "8.8.8.8:443", tlsConfig, quicConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.CloseWithError(0, ""); err != nil {
		t.Fatal(err)
	}
}

func TestQUICDialerLoggerFailure(t *testing.T) {
	expected := errors.New("mocked error")
	d := &QUICDialerLogger{
		Dialer: &netxmocks.QUICContextDialer{
			MockDialContext: func(ctx context.Context, network string,
				address string, tlsConfig *tls.Config,
				quicConfig *quic.Config) (quic.EarlySession, error) {
				return nil, expected
			},
		},
		Logger: log.Log,
	}
	ctx := context.Background()
	tlsConfig := &tls.Config{}
	quicConfig := &quic.Config{}
	sess, err := d.DialContext(ctx, "udp", "8.8.8.8:443", tlsConfig, quicConfig)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if sess != nil {
		t.Fatal("expected nil session")
	}
}
