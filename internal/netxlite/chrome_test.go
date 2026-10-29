package netxlite

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestNewTLSHandshakerChrome(t *testing.T) {
	t.Run("it handshakes and reports the connection state", func(t *testing.T) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			DNSNames:     []string{"www.example.com"},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
		}
		der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
		if err != nil {
			t.Fatal(err)
		}
		ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
			Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
			NextProtos:   []string{"h2"},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		go func() {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
			_ = conn.(*tls.Conn).Handshake()
		}()

		conn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		netx := &Netx{}
		thx := netx.NewTLSHandshakerChrome(model.DiscardLogger)
		tlsConn, err := thx.Handshake(context.Background(), conn, &tls.Config{
			InsecureSkipVerify: true, // #nosec G402 - self-signed test server
			NextProtos:         []string{"h2", "http/1.1"},
			ServerName:         "www.example.com",
		})
		if err != nil {
			t.Fatal(err)
		}
		defer tlsConn.Close()
		state := tlsConn.ConnectionState()
		if state.NegotiatedProtocol != "h2" || state.CurveID != tls.X25519MLKEM768 || !state.HandshakeComplete {
			t.Fatalf("unexpected state: %+v", state)
		}
		if tlsConn.NetConn() != conn {
			t.Fatal("NetConn isn't the TCP connection")
		}
	})

	t.Run("it refuses config fields it doesn't support", func(t *testing.T) {
		_, err := newChromeConn(&mocksConn{}, &tls.Config{MinVersion: tls.VersionTLS13})
		if !errors.Is(err, errUTLSIncompatibleStdlibConfig) {
			t.Fatal("unexpected error", err)
		}
	})
}

// mocksConn is a net.Conn that newChromeConn never uses.
type mocksConn struct {
	net.Conn
}
