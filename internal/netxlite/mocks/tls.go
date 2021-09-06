package mocks

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"reflect"
	"testing"
)

func TestTLSHandshakerHandshake(t *testing.T) {
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
}

// TLSHandshaker is a mockable TLS handshaker.
type TLSHandshaker struct {
	MockHandshake func(ctx context.Context, conn net.Conn, config *tls.Config) (
		net.Conn, tls.ConnectionState, error)
}

// Handshake calls MockHandshake.
func (th *TLSHandshaker) Handshake(ctx context.Context, conn net.Conn, config *tls.Config) (
	net.Conn, tls.ConnectionState, error) {
	return th.MockHandshake(ctx, conn, config)
}

// TLSConn allows to mock netxlite.TLSConn.
type TLSConn struct {
	// Conn is the embedded mockable Conn.
	Conn

	// MockConnectionState allows to mock the ConnectionState method.
	MockConnectionState func() tls.ConnectionState

	// MockHandshakeContext allows to mock the HandshakeContext method.
	MockHandshakeContext func(ctx context.Context) error
}

// ConnectionState calls MockConnectionState.
func (c *TLSConn) ConnectionState() tls.ConnectionState {
	return c.MockConnectionState()
}

// HandshakeContext calls MockHandshakeContext.
func (c *TLSConn) HandshakeContext(ctx context.Context) error {
	return c.MockHandshakeContext(ctx)
}

// TLSDialer allows to mock netxlite.TLSDialer.
type TLSDialer struct {
	MockCloseIdleConnections func()
	MockDialTLSContext       func(ctx context.Context, network, address string) (net.Conn, error)
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (d *TLSDialer) CloseIdleConnections() {
	d.MockCloseIdleConnections()
}

// DialTLSContext calls MockDialTLSContext.
func (d *TLSDialer) DialTLSContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.MockDialTLSContext(ctx, network, address)
}
