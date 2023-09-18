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

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

/*
Test cases:
- Get tlsHandshakeFunc with options
- Apply tlsHandshakeFunc:
  - with EOF
  - with invalid address
  - with success
  - with sni
  - with options
*/
func TestTLSHandshake(t *testing.T) {
	t.Run("Get tlsHandshakeFunc with options", func(t *testing.T) {
		certpool := x509.NewCertPool()
		certpool.AddCert(&x509.Certificate{})

		f := TLSHandshake(
			&ConnPool{},
			TLSHandshakeOptionInsecureSkipVerify(true),
			TLSHandshakeOptionNextProto([]string{"h2"}),
			TLSHandshakeOptionServerName("sni"),
			TLSHandshakeOptionRootCAs(certpool),
		)
		var handshakeFunc *tlsHandshakeFunc
		var ok bool
		if handshakeFunc, ok = f.(*tlsHandshakeFunc); !ok {
			t.Fatal("unexpected type. Expected: tlsHandshakeFunc")
		}
		if !handshakeFunc.InsecureSkipVerify {
			t.Fatalf("unexpected %s, expected %v, got %v", "InsecureSkipVerify", true, false)
		}
		if len(handshakeFunc.NextProto) != 1 || handshakeFunc.NextProto[0] != "h2" {
			t.Fatalf("unexpected %s, expected %v, got %v", "NextProto", []string{"h2"}, handshakeFunc.NextProto)
		}
		if handshakeFunc.ServerName != "sni" {
			t.Fatalf("unexpected %s, expected %s, got %s", "ServerName", "sni", handshakeFunc.ServerName)
		}
		if !handshakeFunc.RootCAs.Equal(certpool) {
			t.Fatalf("unexpected %s, expected %v, got %v", "RootCAs", certpool, handshakeFunc.RootCAs)
		}
	})

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
		tlsConn := &mocks.TLSConn{Conn: tcpConn}

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
				pool := &ConnPool{}
				tlsHandshake := &tlsHandshakeFunc{
					NextProto:  tt.config.nextProtos,
					Pool:       pool,
					ServerName: tt.config.sni,
					handshaker: tt.handshaker,
				}
				idGen := &atomic.Int64{}
				zeroTime := time.Time{}
				trace := measurexlite.NewTrace(idGen.Add(1), zeroTime)
				address := tt.config.address
				if address == "" {
					address = "1.2.3.4:567"
				}
				tcpConn := TCPConnection{
					Address:     address,
					Conn:        &tcpConn,
					IDGenerator: idGen,
					Logger:      model.DiscardLogger,
					Network:     "tcp",
					Trace:       trace,
					ZeroTime:    zeroTime,
				}
				res := tlsHandshake.Apply(context.Background(), &tcpConn)
				if res.Error != tt.expectErr {
					t.Fatalf("unexpected error: %s", res.Error)
				}
				if res.State.Conn != tt.expectConn {
					t.Fatalf("unexpected conn %v", res.State.Conn)
				}
				pool.Close()
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
- With input SNI
- With input domain
- With input host address
- With input IP address
*/
func TestServerNameTLS(t *testing.T) {
	t.Run("With input SNI", func(t *testing.T) {
		sni := "sni"
		tcpConn := TCPConnection{
			Address: "example.com:123",
			Logger:  model.DiscardLogger,
		}
		f := &tlsHandshakeFunc{
			Pool:       &ConnPool{},
			ServerName: sni,
		}
		serverName := f.serverName(&tcpConn)
		if serverName != sni {
			t.Fatalf("unexpected server name: %s", serverName)
		}
	})
	t.Run("With input domain", func(t *testing.T) {
		domain := "domain"
		tcpConn := TCPConnection{
			Address: "example.com:123",
			Domain:  domain,
			Logger:  model.DiscardLogger,
		}
		f := &tlsHandshakeFunc{
			Pool: &ConnPool{},
		}
		serverName := f.serverName(&tcpConn)
		if serverName != domain {
			t.Fatalf("unexpected server name: %s", serverName)
		}
	})
	t.Run("With input host address", func(t *testing.T) {
		hostaddr := "example.com"
		tcpConn := TCPConnection{
			Address: hostaddr + ":123",
			Logger:  model.DiscardLogger,
		}
		f := &tlsHandshakeFunc{
			Pool: &ConnPool{},
		}
		serverName := f.serverName(&tcpConn)
		if serverName != hostaddr {
			t.Fatalf("unexpected server name: %s", serverName)
		}
	})
	t.Run("With input IP address", func(t *testing.T) {
		ip := "1.1.1.1"
		tcpConn := TCPConnection{
			Address: ip,
			Logger:  model.DiscardLogger,
		}
		f := &tlsHandshakeFunc{
			Pool: &ConnPool{},
		}
		serverName := f.serverName(&tcpConn)
		if serverName != "" {
			t.Fatalf("unexpected server name: %s", serverName)
		}
	})
}

// Make sure we get a valid handshaker if no mocked handshaker is configured
func TestHandshakerOrDefault(t *testing.T) {
	f := &tlsHandshakeFunc{
		InsecureSkipVerify: false,
		NextProto:          []string{},
		Pool:               &ConnPool{},
		RootCAs:            &x509.CertPool{},
		ServerName:         "",
		handshaker:         nil,
	}
	handshaker := f.handshakerOrDefault(measurexlite.NewTrace(0, time.Now()), model.DiscardLogger)
	if handshaker == nil {
		t.Fatal("expected non-nil handshaker here")
	}
}
