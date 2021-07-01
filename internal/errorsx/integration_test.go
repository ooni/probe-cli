package errorsx_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestErrorWrapperQUICDialerInvalidCertificate(t *testing.T) {
	nextprotos := []string{"h3"}
	servername := "example.com"
	tlsConf := &tls.Config{
		NextProtos: nextprotos,
		ServerName: servername,
	}

	dlr := &errorsx.ErrorWrapperQUICDialer{Dialer: &netxlite.QUICDialerQUICGo{
		QUICListener: &netxlite.QUICListenerStdlib{},
	}}
	// use Google IP
	sess, err := dlr.DialContext(context.Background(), "udp",
		"216.58.212.164:443", tlsConf, &quic.Config{})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if sess != nil {
		t.Fatal("expected nil sess here")
	}
	if err.Error() != errorsx.FailureSSLInvalidCertificate {
		t.Fatal("unexpected failure")
	}
}

func TestErrorWrapperQUICDialerSuccess(t *testing.T) {
	ctx := context.Background()
	tlsConf := &tls.Config{
		NextProtos: []string{"h3"},
		ServerName: "www.google.com",
	}
	d := &errorsx.ErrorWrapperQUICDialer{Dialer: &netxlite.QUICDialerQUICGo{
		QUICListener: &netxlite.QUICListenerStdlib{},
	}}
	sess, err := d.DialContext(ctx, "udp", "216.58.212.164:443", tlsConf, &quic.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if sess == nil {
		t.Fatal("expected non-nil sess here")
	}
}
