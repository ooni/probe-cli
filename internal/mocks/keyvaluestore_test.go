package mocks

import (
	"errors"
	"testing"
)

func TestKeyValueStore(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		expect := errors.New("mocked error")
		kvs := &KeyValueStore{
			MockGet: func(key string) (value []byte, err error) {
				return nil, expect
			},
		}
		out, err := kvs.Get("antani")
		if !errors.Is(err, expect) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})

	t.Run("Set", func(t *testing.T) {
		expect := errors.New("mocked error")
		kvs := &KeyValueStore{
			MockSet: func(key string, value []byte) (err error) {
				return expect
			},
		}
		err := kvs.Set("antani", nil)
		if !errors.Is(err, expect) {
			t.Fatal("unexpected err", err)
		}
	})
}
