package dslx

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

type tlsTest struct {
	expectedConn net.Conn
	expectedErr  error
	wasClosed    bool
	name         string
	handshaker   *mocks.TLSHandshaker
	tcpConn      mocks.Conn
	sni          string
	address      string
	domain       string
}

func TestTLSHandshake(t *testing.T) {
	// TODO(kelmenhorst): this should be more elaborate
	_ = TLSHandshake(
		&ConnPool{},
		TLSHandshakeOptionInsecureSkipVerify(true),
		TLSHandshakeOptionNextProto([]string{"h3"}),
		TLSHandshakeOptionServerName("sni"),
	)
}

func TestApplyTLS(t *testing.T) {
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
			expectedConn: nil,
			expectedErr:  io.EOF,
			handshaker:   eofHandshaker,
			tcpConn:      tcpConn,
			domain:       "domain.com",
		},
		{
			name:         "success",
			expectedConn: tlsConn,
			expectedErr:  nil,
			wasClosed:    true,
			handshaker:   goodHandshaker,
			tcpConn:      tcpConn,
		},
		{
			name:         "with sni",
			expectedConn: tlsConn,
			expectedErr:  nil,
			wasClosed:    true,
			handshaker:   goodHandshaker,
			tcpConn:      tcpConn,
			sni:          "sni.com",
		},
		{
			name:         "with invalid address",
			address:      "#",
			expectedConn: tlsConn,
			expectedErr:  nil,
			wasClosed:    true,
			handshaker:   goodHandshaker,
			tcpConn:      tcpConn,
		},
	}

	for _, test := range tests {
		pool := &ConnPool{}
		tlsHandshake := &tlsHandshakeFunc{
			Pool:       pool,
			ServerName: test.sni,
			handshaker: test.handshaker,
		}
		idGen := &atomic.Int64{}
		zeroTime := time.Time{}
		trace := measurexlite.NewTrace(idGen.Add(1), zeroTime)
		address := test.address
		if address == "" {
			address = "1.2.3.4:567"
		}
		tcpConn := TCPConnection{
			Address:     address,
			Conn:        &test.tcpConn,
			Domain:      test.domain,
			Network:     "tcp",
			IDGenerator: idGen,
			Logger:      model.DiscardLogger,
			ZeroTime:    zeroTime,
			Trace:       trace,
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
