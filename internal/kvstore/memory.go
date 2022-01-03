package kvstore

import (
	"errors"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// ErrNoSuchKey indicates that there's no value for the given key.
var ErrNoSuchKey = errors.New("no such key")

// Memory is an in-memory key-value store.
type Memory struct {
	// m is the underlying map.
	m map[string][]byte

	// mu provides mutual exclusion
	mu sync.Mutex
}

var _ model.KeyValueStore = &Memory{}

// Get returns the specified key's value. In case of error, the
// error type is such that errors.Is(err, ErrNoSuchKey).
func (kvs *Memory) Get(key string) ([]byte, error) {
	kvs.mu.Lock()
	defer kvs.mu.Unlock()
	value, ok := kvs.m[key]
	if !ok {
		return nil, ErrNoSuchKey
	}
	return value, nil
}

// Set sets a key into the key-value store.
func (kvs *Memory) Set(key string, value []byte) error {
	kvs.mu.Lock()
	defer kvs.mu.Unlock()
	if kvs.m == nil {
		kvs.m = make(map[string][]byte)
	}
	kvs.m[key] = value
	return nil
}
