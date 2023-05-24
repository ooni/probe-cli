package mocks

import "github.com/ooni/probe-cli/v3/internal/model"

// KeyValueStore allows mocking model.KeyValueStore.
type KeyValueStore struct {
	MockGet func(key string) (value []byte, err error)

	MockSet func(key string, value []byte) (err error)
}

var _ model.KeyValueStore = &KeyValueStore{}

func (kvs *KeyValueStore) Get(key string) (value []byte, err error) {
	return kvs.MockGet(key)
}

func (kvs *KeyValueStore) Set(key string, value []byte) (err error) {
	return kvs.MockSet(key, value)
}
