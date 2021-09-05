package mocks

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestDialerDialContext(t *testing.T) {
	expected := errors.New("mocked error")
	d := Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return nil, expected
		},
	}
	ctx := context.Background()
	conn, err := d.DialContext(ctx, "tcp", "8.8.8.8:53")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn")
	}
}

func TestDialerCloseIdleConnections(t *testing.T) {
	var called bool
	d := &Dialer{
		MockCloseIdleConnections: func() {
			called = true
		},
	}
	d.CloseIdleConnections()
	if !called {
		t.Fatal("not called")
	}
}
