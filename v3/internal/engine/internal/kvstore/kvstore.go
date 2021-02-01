// Package kvstore contains key-value stores
package kvstore

import (
	"errors"
	"sync"
)

// MemoryKeyValueStore is an in-memory key-value store
type MemoryKeyValueStore struct {
	m  map[string][]byte
	mu sync.Mutex
}

// NewMemoryKeyValueStore creates a new in-memory key-value store
func NewMemoryKeyValueStore() *MemoryKeyValueStore {
	return &MemoryKeyValueStore{
		m: make(map[string][]byte),
	}
}

// Get returns a key from the key value store
func (kvs *MemoryKeyValueStore) Get(key string) ([]byte, error) {
	var (
		err   error
		ok    bool
		value []byte
	)
	kvs.mu.Lock()
	defer kvs.mu.Unlock()
	value, ok = kvs.m[key]
	if !ok {
		err = errors.New("no such key")
	}
	return value, err
}

// Set sets a key into the key value store
func (kvs *MemoryKeyValueStore) Set(key string, value []byte) error {
	kvs.mu.Lock()
	defer kvs.mu.Unlock()
	kvs.m[key] = value
	return nil
}
