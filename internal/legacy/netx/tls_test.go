package netx

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/legacy/tracex"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestNewTLSDialer(t *testing.T) {
	t.Run("we always have error wrapping", func(t *testing.T) {
		server := testingx.MustNewTLSServer(testingx.TLSHandlerReset())
		defer server.Close()
		tdx := NewTLSDialer(Config{})
		conn, err := tdx.DialTLSContext(context.Background(), "tcp", server.Endpoint())
		if err == nil || err.Error() != netxlite.FailureConnectionReset {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("we can collect measurements", func(t *testing.T) {
		server := testingx.MustNewTLSServer(testingx.TLSHandlerReset())
		defer server.Close()
		saver := &tracex.Saver{}
		tdx := NewTLSDialer(Config{
			Saver: saver,
		})
		conn, err := tdx.DialTLSContext(context.Background(), "tcp", server.Endpoint())
		if err == nil || err.Error() != netxlite.FailureConnectionReset {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		if len(saver.Read()) <= 0 {
			t.Fatal("did not read any event")
		}
	})

	t.Run("we can skip TLS verification", func(t *testing.T) {
		ca := netem.MustNewCA()
		cert := ca.MustNewTLSCertificate("www.example.com")
		server := testingx.MustNewTLSServer(testingx.TLSHandlerHandshakeAndWriteText(cert, testingx.HTTPBlockpage451))
		defer server.Close()
		tdx := NewTLSDialer(Config{TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		}})
		conn, err := tdx.DialTLSContext(context.Background(), "tcp", server.Endpoint())
		if err != nil {
			t.Fatal(err.(*netxlite.ErrWrapper).WrappedErr)
		}
		conn.Close()
	})

	t.Run("we can set the cert pool", func(t *testing.T) {
		ca := netem.MustNewCA()
		cert := ca.MustNewTLSCertificate("dns.google")
		server := testingx.MustNewTLSServer(testingx.TLSHandlerHandshakeAndWriteText(cert, testingx.HTTPBlockpage451))
		defer server.Close()
		tdx := NewTLSDialer(Config{
			TLSConfig: &tls.Config{
				RootCAs:    ca.DefaultCertPool(),
				ServerName: "dns.google",
			},
		})
		conn, err := tdx.DialTLSContext(context.Background(), "tcp", server.Endpoint())
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	})
}
