package testingsocks5

import (
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestClientErrorPaths(t *testing.T) {
	t.Run("conn.Write fails", func(t *testing.T) {
		expected := errors.New("mocked error")
		conn := &mocks.Conn{
			MockWrite: func(b []byte) (int, error) {
				return 0, expected
			},
		}
		c := &client{
			exchanges: []exchange{{
				send:   []byte{1, 2, 3, 4},
				expect: []byte{},
			}},
		}
		err := c.run(model.DiscardLogger, conn)
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("conn.Read fails", func(t *testing.T) {
		expected := errors.New("mocked error")
		conn := &mocks.Conn{
			MockWrite: func(b []byte) (int, error) {
				return len(b), nil
			},
			MockRead: func(b []byte) (int, error) {
				return 0, expected
			},
		}
		c := &client{
			exchanges: []exchange{{
				send:   []byte{1, 2, 3, 4},
				expect: []byte{4, 3, 2, 1},
			}},
		}
		err := c.run(model.DiscardLogger, conn)
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("when we get an unexpected response", func(t *testing.T) {
		conn := &mocks.Conn{
			MockWrite: func(b []byte) (int, error) {
				return len(b), nil
			},
			MockRead: func(b []byte) (int, error) {
				runtimex.Assert(len(b) == 4, "unexpected buffer length")
				copy(b, []byte{1, 2, 3, 4})
				return len(b), nil
			},
		}
		c := &client{
			exchanges: []exchange{{
				send:   []byte{1, 2, 3, 4},
				expect: []byte{4, 3, 2, 1},
			}},
		}
		err := c.run(model.DiscardLogger, conn)
		if !errors.Is(err, errUnexpectedResponse) {
			t.Fatal("unexpected error", err)
		}
	})
}
