package mocks

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"reflect"
	"testing"
)

func TestTLSHandshaker(t *testing.T) {
	t.Run("Handshake", func(t *testing.T) {
		expected := errors.New("mocked error")
		conn := &Conn{}
		ctx := context.Background()
		config := &tls.Config{}
		th := &TLSHandshaker{
			MockHandshake: func(ctx context.Context, conn net.Conn,
				config *tls.Config) (net.Conn, tls.ConnectionState, error) {
				return nil, tls.ConnectionState{}, expected
			},
		}
		tlsConn, connState, err := th.Handshake(ctx, conn, config)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if !reflect.ValueOf(connState).IsZero() {
			t.Fatal("expected zero ConnectionState here")
		}
		if tlsConn != nil {
			t.Fatal("expected nil conn here")
		}
	})
}

func TestTLSConn(t *testing.T) {
	t.Run("ConnectionState", func(t *testing.T) {
		state := tls.ConnectionState{Version: tls.VersionTLS12}
		c := &TLSConn{
			MockConnectionState: func() tls.ConnectionState {
				return state
			},
		}
		out := c.ConnectionState()
		if !reflect.DeepEqual(out, state) {
			t.Fatal("not the result we expected")
		}
	})

	t.Run("HandshakeContext", func(t *testing.T) {
		expected := errors.New("mocked error")
		c := &TLSConn{
			MockHandshakeContext: func(ctx context.Context) error {
				return expected
			},
		}
		err := c.HandshakeContext(context.Background())
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
	})

	t.Run("NetConn", func(t *testing.T) {
		conn := &Conn{}
		c := &TLSConn{
			MockNetConn: func() net.Conn {
				return conn
			},
		}
		if o := c.NetConn(); o != conn {
			t.Fatal("unexpected result")
		}
	})
}

func TestTLSDialer(t *testing.T) {
	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		td := &TLSDialer{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		td.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("DialTLSContext", func(t *testing.T) {
		expected := errors.New("mocked error")
		td := &TLSDialer{
			MockDialTLSContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		conn, err := td.DialTLSContext(ctx, "", "")
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn here")
		}
	})
}
