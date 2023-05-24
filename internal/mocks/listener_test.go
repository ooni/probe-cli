package mocks

import (
	"errors"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestListener(t *testing.T) {
	t.Run("Accept", func(t *testing.T) {
		expected := errors.New("mocked error")
		li := &Listener{
			MockAccept: func() (net.Conn, error) {
				return nil, expected
			},
		}
		conn, err := li.Accept()
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("Close", func(t *testing.T) {
		expected := errors.New("mocked error")
		li := &Listener{
			MockClose: func() error {
				return expected
			},
		}
		err := li.Close()
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("Addr", func(t *testing.T) {
		addr := &net.TCPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 1234,
		}
		li := &Listener{
			MockAddr: func() net.Addr {
				return addr
			},
		}
		outAddr := li.Addr()
		if diff := cmp.Diff(addr, outAddr); diff != "" {
			t.Fatal(diff)
		}
	})
}
