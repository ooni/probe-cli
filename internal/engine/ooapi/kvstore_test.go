package ooapi

import (
	"errors"
	"fmt"
	"sync"
)

var errMemkvstoreNotFound = errors.New("memkvstore: not found")

type MemKVStore struct {
	m  map[string][]byte
	mu sync.Mutex
}

func (kvs *MemKVStore) Get(key string) ([]byte, error) {
	defer kvs.mu.Unlock()
	kvs.mu.Lock()
	out, good := kvs.m[key]
	if !good {
		return nil, fmt.Errorf("%w: %s", errMemkvstoreNotFound, key)
	}
	return out, nil
}

func (kvs *MemKVStore) Set(key string, value []byte) error {
	defer kvs.mu.Unlock()
	kvs.mu.Lock()
	if kvs.m == nil {
		kvs.m = make(map[string][]byte)
	}
	kvs.m[key] = value
	return nil
}
