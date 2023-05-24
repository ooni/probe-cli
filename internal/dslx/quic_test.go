package dslx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

/*
Test cases:
- Get quicHandshakeFunc with options
- Apply quicHandshakeFunc:
  - with EOF
  - success
  - with sni
*/
func TestQUICHandshake(t *testing.T) {
	t.Run("Get quicHandshakeFunc with options", func(t *testing.T) {
		certpool := x509.NewCertPool()
		certpool.AddCert(&x509.Certificate{})

		f := QUICHandshake(
			&ConnPool{},
			QUICHandshakeOptionInsecureSkipVerify(true),
			QUICHandshakeOptionServerName("sni"),
			QUICHandshakeOptionRootCAs(certpool),
		)
		if _, ok := f.(*quicHandshakeFunc); !ok {
			t.Fatal("unexpected type. Expected: quicHandshakeFunc")
		}
	})

	t.Run("Apply quicHandshakeFunc", func(t *testing.T) {
		wasClosed := false
		plainConn := &mocks.QUICEarlyConnection{
			MockCloseWithError: func(code quic.ApplicationErrorCode, reason string) error {
				wasClosed = true
				return nil
			},
			MockConnectionState: func() quic.ConnectionState {
				return quic.ConnectionState{}
			},
		}

		eofDialer := &mocks.QUICDialer{
			MockDialContext: func(ctx context.Context, address string, tlsConfig *tls.Config,
				quicConfig *quic.Config) (quic.EarlyConnection, error) {
				return nil, io.EOF
			},
		}

		goodDialer := &mocks.QUICDialer{
			MockDialContext: func(ctx context.Context, address string, tlsConfig *tls.Config,
				quicConfig *quic.Config) (quic.EarlyConnection, error) {
				return plainConn, nil
			},
		}

		tests := map[string]struct {
			dialer     model.QUICDialer
			sni        string
			expectConn quic.EarlyConnection
			expectErr  error
			closed     bool
		}{
			"with EOF": {expectConn: nil, expectErr: io.EOF, closed: false, dialer: eofDialer},
			"success":  {expectConn: plainConn, expectErr: nil, closed: true, dialer: goodDialer},
			"with sni": {expectConn: plainConn, expectErr: nil, closed: true, dialer: goodDialer, sni: "sni.com"},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				pool := &ConnPool{}
				quicHandshake := &quicHandshakeFunc{Pool: pool, dialer: tt.dialer, ServerName: tt.sni}
				endpoint := &Endpoint{
					Address:     "1.2.3.4:567",
					Network:     "udp",
					IDGenerator: &atomic.Int64{},
					Logger:      model.DiscardLogger,
					ZeroTime:    time.Time{},
				}
				res := quicHandshake.Apply(context.Background(), endpoint)
				if res.Error != tt.expectErr {
					t.Fatalf("unexpected error: %s", res.Error)
				}
				if res.State.QUICConn != tt.expectConn {
					t.Fatalf("unexpected conn: %s", res.State.QUICConn)
				}
				pool.Close()
				if wasClosed != tt.closed {
					t.Fatalf("unexpected connection closed state: %v", wasClosed)
				}
			})
			wasClosed = false
		}

		t.Run("with nil dialer", func(t *testing.T) {
			quicHandshake := &quicHandshakeFunc{Pool: &ConnPool{}, dialer: nil}
			endpoint := &Endpoint{
				Address:     "1.2.3.4:567",
				Network:     "udp",
				IDGenerator: &atomic.Int64{},
				Logger:      model.DiscardLogger,
				ZeroTime:    time.Time{},
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			res := quicHandshake.Apply(ctx, endpoint)

			if res.Error == nil {
				t.Fatalf("expected an error here")
			}
			if res.State.QUICConn != nil {
				t.Fatalf("unexpected conn: %s", res.State.QUICConn)
			}
		})
	})
}

/*
Test cases:
- With input SNI
- With input domain
- With input host address
- With input IP address
*/
func TestServerNameQUIC(t *testing.T) {
	t.Run("With input SNI", func(t *testing.T) {
		sni := "sni"
		endpoint := &Endpoint{
			Address: "example.com:123",
			Logger:  model.DiscardLogger,
		}
		f := &quicHandshakeFunc{Pool: &ConnPool{}, ServerName: sni}
		serverName := f.serverName(endpoint)
		if serverName != sni {
			t.Fatalf("unexpected server name: %s", serverName)
		}
	})
	t.Run("With input domain", func(t *testing.T) {
		domain := "domain"
		endpoint := &Endpoint{
			Address: "example.com:123",
			Domain:  domain,
			Logger:  model.DiscardLogger,
		}
		f := &quicHandshakeFunc{Pool: &ConnPool{}}
		serverName := f.serverName(endpoint)
		if serverName != domain {
			t.Fatalf("unexpected server name: %s", serverName)
		}
	})
	t.Run("With input host address", func(t *testing.T) {
		hostaddr := "example.com"
		endpoint := &Endpoint{
			Address: hostaddr + ":123",
			Logger:  model.DiscardLogger,
		}
		f := &quicHandshakeFunc{Pool: &ConnPool{}}
		serverName := f.serverName(endpoint)
		if serverName != hostaddr {
			t.Fatalf("unexpected server name: %s", serverName)
		}
	})
	t.Run("With input IP address", func(t *testing.T) {
		ip := "1.1.1.1"
		endpoint := &Endpoint{
			Address: ip,
			Logger:  model.DiscardLogger,
		}
		f := &quicHandshakeFunc{Pool: &ConnPool{}}
		serverName := f.serverName(endpoint)
		if serverName != "" {
			t.Fatalf("unexpected server name: %s", serverName)
		}
	})
}
