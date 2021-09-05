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
