package badproxy

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/martian/v3/mitm"
)

func TestCleartext(t *testing.T) {
	listener := newproxy(t)
	checkdial(t, listener.Addr().String(), nil, net.Dial)
	killproxy(t, listener)
}

func TestTLS(t *testing.T) {
	listener := newproxytls(t)
	checkdial(t, listener.Addr().String(), nil,
		func(network, address string) (net.Conn, error) {
			conn, err := tls.Dial(network, address, &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         "antani.local",
			})
			if err != nil {
				return nil, err
			}
			if err = conn.Handshake(); err != nil {
				conn.Close()
				return nil, err
			}
			return conn, nil
		})
	killproxy(t, listener)
}

func TestListenError(t *testing.T) {
	proxy := NewCensoringProxy()
	listener, err := proxy.Start("8.8.8.8:80")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if listener != nil {
		t.Fatal("expected nil listener here")
	}
}

func TestStarTLS(t *testing.T) {
	expected := errors.New("mocked error")

	t.Run("when we cannot create a new authority", func(t *testing.T) {
		proxy := NewCensoringProxy()
		proxy.mitmNewAuthority = func(
			name string, organization string,
			validity time.Duration,
		) (*x509.Certificate, *rsa.PrivateKey, error) {
			return nil, nil, expected
		}
		cert, privkey, err := proxy.StartTLS("127.0.0.1:0")
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
		if cert != nil {
			t.Fatal("expected nil cert")
		}
		if privkey != nil {
			t.Fatal("expected nil privkey")
		}
	})

	t.Run("when we cannot create a new config", func(t *testing.T) {
		proxy := NewCensoringProxy()
		proxy.mitmNewConfig = func(
			ca *x509.Certificate, privateKey interface{},
		) (*mitm.Config, error) {
			return nil, expected
		}
		cert, privkey, err := proxy.StartTLS("127.0.0.1:0")
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
		if cert != nil {
			t.Fatal("expected nil cert")
		}
		if privkey != nil {
			t.Fatal("expected nil privkey")
		}
	})

	t.Run("when we cannot listen", func(t *testing.T) {
		proxy := NewCensoringProxy()
		proxy.tlsListen = func(
			network string, laddr string, config *tls.Config,
		) (net.Listener, error) {
			return nil, expected
		}
		cert, privkey, err := proxy.StartTLS("127.0.0.1:0")
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
		if cert != nil {
			t.Fatal("expected nil cert")
		}
		if privkey != nil {
			t.Fatal("expected nil privkey")
		}
	})
}

func newproxy(t *testing.T) net.Listener {
	proxy := NewCensoringProxy()
	listener, err := proxy.Start("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return listener
}

func newproxytls(t *testing.T) net.Listener {
	proxy := NewCensoringProxy()
	listener, _, err := proxy.StartTLS("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return listener
}

func killproxy(t *testing.T, listener net.Listener) {
	err := listener.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func checkdial(
	t *testing.T, proxyAddr string, expectErr error,
	dial func(network, address string) (net.Conn, error),
) {
	conn, err := dial("tcp", proxyAddr)
	if err != expectErr {
		t.Fatal("not the result we expected")
	}
	if conn == nil && expectErr == nil {
		t.Fatal("expected actionable conn")
	}
	if conn != nil && expectErr != nil {
		t.Fatal("expected nil conn")
	}
	if conn != nil {
		conn.Write([]byte("123454321"))
		conn.Close()
	}
}
