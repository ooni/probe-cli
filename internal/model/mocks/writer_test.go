package mocks

import (
	"errors"
	"testing"
)

func TestWriter(t *testing.T) {
	t.Run("Write", func(t *testing.T) {
		expected := errors.New("mocked error")
		r := &Writer{
			MockWrite: func(b []byte) (int, error) {
				return 0, expected
			},
		}
		b := make([]byte, 128)
		count, err := r.Write(b)
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
		if count != 0 {
			t.Fatal("unexpected count", count)
		}
	})
}
