package filtering

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestTLSServer(t *testing.T) {
	t.Run("TLSActionReset", func(t *testing.T) {
		srv := NewTLSServer(TLSActionReset)
		defer srv.Close()
		config := &tls.Config{ServerName: "dns.google"}
		conn, err := tls.Dial("tcp", srv.Endpoint(), config)
		if !errors.Is(err, syscall.ECONNRESET) {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("TLSActionTimeout", func(t *testing.T) {
		srv := NewTLSServer(TLSActionTimeout)
		defer srv.Close()
		config := &tls.Config{ServerName: "dns.google"}
		d := &tls.Dialer{Config: config}
		ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
		defer cancel()
		conn, err := d.DialContext(ctx, "tcp", srv.Endpoint())
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("TLSActionAlertInternalError", func(t *testing.T) {
		srv := NewTLSServer(TLSActionAlertInternalError)
		defer srv.Close()
		config := &tls.Config{ServerName: "dns.google"}
		conn, err := tls.Dial("tcp", srv.Endpoint(), config)
		if err == nil || !strings.HasSuffix(err.Error(), "tls: internal error") {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("TLSActionAlertUnrecognizedName", func(t *testing.T) {
		srv := NewTLSServer(TLSActionAlertUnrecognizedName)
		defer srv.Close()
		config := &tls.Config{ServerName: "dns.google"}
		conn, err := tls.Dial("tcp", srv.Endpoint(), config)
		if err == nil || !strings.HasSuffix(err.Error(), "tls: unrecognized name") {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("TLSActionEOF", func(t *testing.T) {
		srv := NewTLSServer(TLSActionEOF)
		defer srv.Close()
		config := &tls.Config{ServerName: "dns.google"}
		conn, err := tls.Dial("tcp", srv.Endpoint(), config)
		if !errors.Is(err, io.EOF) {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("TLSActionBlockText", func(t *testing.T) {
		t.Run("certificate error when we're validating", func(t *testing.T) {
			srv := NewTLSServer(TLSActionBlockText)
			defer srv.Close()
			//     Certificate.Verify now uses platform APIs to verify certificate validity
			//     on macOS and iOS when it is called with a nil VerifyOpts.Roots or when using
			//     the root pool returned from SystemCertPool. "
			//
			//     -- https://tip.golang.org/doc/go1.18
			//
			// So we need to explicitly use our default cert pool otherwise we will
			// see this test failing with a different error string here.
			config := &tls.Config{
				ServerName: "dns.google",
				RootCAs:    netxlite.NewDefaultCertPool(),
			}
			conn, err := tls.Dial("tcp", srv.Endpoint(), config)
			if err == nil || !strings.HasSuffix(err.Error(), "certificate signed by unknown authority") {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})

		t.Run("blocktext when we skip validation", func(t *testing.T) {
			srv := NewTLSServer(TLSActionBlockText)
			defer srv.Close()
			config := &tls.Config{InsecureSkipVerify: true, ServerName: "dns.google"}
			conn, err := tls.Dial("tcp", srv.Endpoint(), config)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			data, err := io.ReadAll(conn)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(HTTPBlockpage451, data) {
				t.Fatal("unexpected block text")
			}
		})

		t.Run("blocktext when we configure the cert pool", func(t *testing.T) {
			srv := NewTLSServer(TLSActionBlockText)
			defer srv.Close()
			config := &tls.Config{RootCAs: srv.CertPool(), ServerName: "dns.google"}
			conn, err := tls.Dial("tcp", srv.Endpoint(), config)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			data, err := io.ReadAll(conn)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(HTTPBlockpage451, data) {
				t.Fatal("unexpected block text")
			}
		})
	})
}
