package mocks

import (
	"context"
	"crypto/tls"
	"errors"
	"reflect"
	"testing"
)

func TestTLSConnConnectionState(t *testing.T) {
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
}

func TestTLSConnHandshakeContext(t *testing.T) {
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
}
