package mocks

import (
	"errors"
	"testing"
)

func TestReader(t *testing.T) {
	t.Run("Read", func(t *testing.T) {
		expected := errors.New("mocked error")
		r := &Reader{
			MockRead: func(b []byte) (int, error) {
				return 0, expected
			},
		}
		b := make([]byte, 128)
		count, err := r.Read(b)
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
		if count != 0 {
			t.Fatal("unexpected count", count)
		}
	})
}
