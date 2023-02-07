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
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestTLSHandshake(t *testing.T) {
	certpool := x509.NewCertPool()
	certpool.AddCert(&x509.Certificate{})

	f := TLSHandshake(
		&ConnPool{},
		TLSHandshakeOptionInsecureSkipVerify(true),
		TLSHandshakeOptionNextProto([]string{"h3"}),
		TLSHandshakeOptionServerName("sni"),
		TLSHandshakeOptionRootCAs(certpool),
	)
	var handshakeFunc *tlsHandshakeFunc
	var ok bool
	if handshakeFunc, ok = f.(*tlsHandshakeFunc); !ok {
		t.Fatal("TLSHandshake: unexpected type. Expected: tlsHandshakeFunc")
	}
	if !handshakeFunc.InsecureSkipVerify {
		t.Fatalf("TLSHandshake: %s, expected %v, got %v", "InsecureSkipVerify", true, false)
	}
	if len(handshakeFunc.NextProto) != 1 || handshakeFunc.NextProto[0] != "h3" {
		t.Fatalf("TLSHandshake: %s, expected %v, got %v", "NextProto", []string{"h3"}, handshakeFunc.NextProto)
	}
	if handshakeFunc.ServerName != "sni" {
		t.Fatalf("TLSHandshake: %s, expected %s, got %s", "ServerName", "sni", handshakeFunc.ServerName)
	}
	if !handshakeFunc.RootCAs.Equal(certpool) {
		t.Fatalf("TLSHandshake: %s, expected %v, got %v", "RootCAs", certpool, handshakeFunc.RootCAs)
	}
}

func TestApplyTLS(t *testing.T) {
	type configOptions struct {
		sni        string
		address    string
		domain     string
		nextProtos []string
	}

	type tlsTest struct {
		expectedConn net.Conn
		expectedErr  error
		wasClosed    bool
		name         string
		handshaker   *mocks.TLSHandshaker
		tcpConn      mocks.Conn
		config       configOptions
	}

	tcpConn := mocks.Conn{
		MockClose: func() error {
			wasClosed = true
			return nil
		},
	}
	tlsConn := &mocks.TLSConn{Conn: tcpConn}
	eofHandshaker := &mocks.TLSHandshaker{
		MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
			return nil, tls.ConnectionState{}, io.EOF
		},
	}
	goodHandshaker := &mocks.TLSHandshaker{
		MockHandshake: func(ctx context.Context, conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
			return tlsConn, tls.ConnectionState{}, nil
		},
	}
	tests := []tlsTest{
		{
			name:         "with EOF",
			tcpConn:      tcpConn,
			handshaker:   eofHandshaker,
			expectedConn: nil,
			expectedErr:  io.EOF,
		},
		{
			name:         "success",
			tcpConn:      tcpConn,
			handshaker:   goodHandshaker,
			expectedConn: tlsConn,
			expectedErr:  nil,
			wasClosed:    true,
		},
		{
			name:         "with sni",
			config:       configOptions{sni: "sni.com"},
			tcpConn:      tcpConn,
			handshaker:   goodHandshaker,
			expectedConn: tlsConn,
			expectedErr:  nil,
			wasClosed:    true,
		},
		{
			name:         "with invalid address",
			config:       configOptions{address: "#"},
			tcpConn:      tcpConn,
			handshaker:   goodHandshaker,
			expectedConn: tlsConn,
			expectedErr:  nil,
			wasClosed:    true,
		},
		{
			name:         "with options",
			config:       configOptions{domain: "domain.com", nextProtos: []string{"h3"}},
			tcpConn:      tcpConn,
			handshaker:   goodHandshaker,
			expectedConn: tlsConn,
			expectedErr:  nil,
			wasClosed:    true,
		},
	}

	for _, test := range tests {
		pool := &ConnPool{}
		tlsHandshake := &tlsHandshakeFunc{
			NextProto:  test.config.nextProtos,
			Pool:       pool,
			ServerName: test.config.sni,
			handshaker: test.handshaker,
		}
		idGen := &atomic.Int64{}
		zeroTime := time.Time{}
		trace := measurexlite.NewTrace(idGen.Add(1), zeroTime)
		address := test.config.address
		if address == "" {
			address = "1.2.3.4:567"
		}
		tcpConn := TCPConnection{
			Address:     address,
			Conn:        &test.tcpConn,
			Domain:      test.config.domain,
			IDGenerator: idGen,
			Logger:      model.DiscardLogger,
			Network:     "tcp",
			Trace:       trace,
			ZeroTime:    zeroTime,
		}
		res := tlsHandshake.Apply(context.Background(), &tcpConn)
		if res.Error != test.expectedErr {
			t.Fatalf("%s: expected error %s, got %s", test.name, test.expectedErr, res.Error)
		}
		if res.State.Conn != test.expectedConn {
			t.Fatalf("%s: expected conn %v, got %v", test.name, test.expectedConn, res.State.Conn)
		}
		pool.Close()
		if wasClosed != test.wasClosed {
			t.Fatalf("%s: expected closeErr %v, got %v", test.name, test.wasClosed, wasClosed)
		}
		wasClosed = false
	}
}
