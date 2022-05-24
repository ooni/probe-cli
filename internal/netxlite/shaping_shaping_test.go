//go:build shaping

package netxlite

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestNewShapingDialerx(t *testing.T) {
	t.Run("failure", func(t *testing.T) {
		expected := errors.New("mocked error")
		d := &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return nil, expected
			},
		}
		shd := NewMaybeShapingDialer(d)
		conn, err := shd.DialContext(context.Background(), "tcp", "8.8.8.8:443")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("success", func(t *testing.T) {
		expected := errors.New("mocked error")
		uc := &mocks.Conn{
			MockRead: func(b []byte) (int, error) {
				return 0, expected
			},
			MockWrite: func(b []byte) (int, error) {
				return 0, expected
			},
		}
		d := &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return uc, nil
			},
		}
		shd := NewMaybeShapingDialer(d)
		conn, err := shd.DialContext(context.Background(), "tcp", "8.8.8.8:443")
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := conn.(*shapingConn); !ok {
			t.Fatal("not shapingConn")
		}
		validateCountAndErr := func(count int, err error) {
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if count != 0 {
				t.Fatal("expected zero")
			}
		}
		validateCountAndErr(conn.Read(make([]byte, 16)))
		validateCountAndErr(conn.Write(make([]byte, 16)))
	})
}
