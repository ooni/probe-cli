package dslx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestTLSNewConfig(t *testing.T) {
	t.Run("without options", func(t *testing.T) {
		config := tlsNewConfig("1.1.1.1:443", []string{"h2", "http/1.1"}, "sni", model.DiscardLogger)

		if config.InsecureSkipVerify {
			t.Fatalf("unexpected %s, expected %v, got %v", "InsecureSkipVerify", false, config.InsecureSkipVerify)
		}
		if diff := cmp.Diff([]string{"h2", "http/1.1"}, config.NextProtos); diff != "" {
			t.Fatal(diff)
		}
		if config.ServerName != "sni" {
			t.Fatalf("unexpected %s, expected %s, got %s", "ServerName", "sni", config.ServerName)
		}
		if !config.RootCAs.Equal(nil) {
			t.Fatalf("unexpected %s, expected %v, got %v", "RootCAs", nil, config.RootCAs)
		}
	})

	t.Run("with options", func(t *testing.T) {
		certpool := x509.NewCertPool()
		certpool.AddCert(&x509.Certificate{})

		config := tlsNewConfig(
			"1.1.1.1:443", []string{"h2", "http/1.1"}, "sni", model.DiscardLogger,
			TLSHandshakeOptionInsecureSkipVerify(true),
			TLSHandshakeOptionNextProto([]string{"h2"}),
			TLSHandshakeOptionServerName("example.domain"),
			TLSHandshakeOptionRootCAs(certpool),
		)

		if !config.InsecureSkipVerify {
			t.Fatalf("unexpected %s, expected %v, got %v", "InsecureSkipVerify", true, config.InsecureSkipVerify)
		}
		if diff := cmp.Diff([]string{"h2"}, config.NextProtos); diff != "" {
			t.Fatal(diff)
		}
		if config.ServerName != "example.domain" {
			t.Fatalf("unexpected %s, expected %s, got %s", "ServerName", "example.domain", config.ServerName)
		}
		if !config.RootCAs.Equal(certpool) {
			t.Fatalf("unexpected %s, expected %v, got %v", "RootCAs", nil, config.RootCAs)
		}
	})
}

/*
Test cases:
- Apply tlsHandshakeFunc:
  - with EOF
  - with invalid address
  - with success
  - with sni
  - with options
*/
func TestTLSHandshake(t *testing.T) {
	t.Run("Apply tlsHandshakeFunc", func(t *testing.T) {
		wasClosed := false

		type configOptions struct {
			sni        string
			address    string
			nextProtos []string
		}
		tcpConn := mocks.Conn{
			MockClose: func() error {
				wasClosed = true
				return nil
			},
		}
		tlsConn := &mocks.TLSConn{
			Conn: tcpConn,
			MockConnectionState: func() tls.ConnectionState {
				return tls.ConnectionState{}
			},
		}

		eofHandshaker := &mocks.TLSHandshaker{
			MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (model.TLSConn, error) {
				return nil, io.EOF
			},
		}

		goodHandshaker := &mocks.TLSHandshaker{
			MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (model.TLSConn, error) {
				return tlsConn, nil
			},
		}

		tests := map[string]struct {
			config     configOptions
			handshaker *mocks.TLSHandshaker
			expectConn net.Conn
			expectErr  error
			closed     bool
		}{
			"with EOF": {
				handshaker: eofHandshaker,
				expectConn: nil,
				expectErr:  io.EOF,
				closed:     false,
			},
			"with invalid address": {
				config:     configOptions{address: "#"},
				handshaker: goodHandshaker,
				expectConn: tlsConn,
				expectErr:  nil,
				closed:     true,
			},
			"with success": {
				handshaker: goodHandshaker,
				expectConn: tlsConn,
				expectErr:  nil,
				closed:     true,
			},
			"with sni": {
				config:     configOptions{sni: "sni.com"},
				handshaker: goodHandshaker,
				expectConn: tlsConn,
				expectErr:  nil,
				closed:     true,
			},
			"with options": {
				config:     configOptions{nextProtos: []string{"h2", "http/1.1"}},
				handshaker: goodHandshaker,
				expectConn: tlsConn,
				expectErr:  nil,
				closed:     true,
			},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				rt := NewMinimalRuntime(model.DiscardLogger, time.Now(), MinimalRuntimeOptionMeasuringNetwork(&mocks.MeasuringNetwork{
					MockNewTLSHandshakerStdlib: func(logger model.DebugLogger) model.TLSHandshaker {
						return tt.handshaker
					},
				}))
				tlsHandshake := TLSHandshake(rt,
					TLSHandshakeOptionNextProto(tt.config.nextProtos),
					TLSHandshakeOptionServerName(tt.config.sni),
				)
				idGen := &atomic.Int64{}
				zeroTime := time.Time{}
				trace := rt.NewTrace(idGen.Add(1), zeroTime)
				address := tt.config.address
				if address == "" {
					address = "1.2.3.4:567"
				}
				tcpConn := TCPConnection{
					Address: address,
					Conn:    &tcpConn,
					Network: "tcp",
					Trace:   trace,
				}
				res := tlsHandshake.Apply(context.Background(), NewMaybeWithValue(&tcpConn))
				if res.Error != tt.expectErr {
					t.Fatalf("unexpected error: %s", res.Error)
				}
				if res.State.Conn != tt.expectConn {
					t.Fatalf("unexpected conn %v", res.State.Conn)
				}
				rt.Close()
				if wasClosed != tt.closed {
					t.Fatalf("unexpected connection closed state %v", wasClosed)
				}
			})
			wasClosed = false
		}
	})
}

/*
Test cases:
- With domain
- With host address
- With IP address
*/
func TestTLSServerName(t *testing.T) {
	t.Run("With domain", func(t *testing.T) {
		serverName := tlsServerName("example.com:123", "domain", model.DiscardLogger)
		if serverName != "domain" {
			t.Fatalf("unexpected server name: %s", serverName)
		}
	})

	t.Run("With host address", func(t *testing.T) {
		serverName := tlsServerName("1.1.1.1:443", "", model.DiscardLogger)
		if serverName != "1.1.1.1" {
			t.Fatalf("unexpected server name: %s", serverName)
		}
	})

	t.Run("With IP address", func(t *testing.T) {
		serverName := tlsServerName("1.1.1.1", "", model.DiscardLogger)
		if serverName != "" {
			t.Fatalf("unexpected server name: %s", serverName)
		}
	})
}
