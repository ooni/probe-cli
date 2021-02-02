package engine

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/rogpeppe/go-internal/lockedfile"
)

// KVStore is a simple, atomic key-value store. The user of
// probe-engine should supply an implementation of this interface,
// which will be used by probe-engine to store specific data.
type KVStore interface {
	Get(key string) (value []byte, err error)
	Set(key string, value []byte) (err error)
}

// FileSystemKVStore is a directory based KVStore
type FileSystemKVStore struct {
	basedir string
}

// NewFileSystemKVStore creates a new FileSystemKVStore.
func NewFileSystemKVStore(basedir string) (kvs *FileSystemKVStore, err error) {
	if err = os.MkdirAll(basedir, 0700); err == nil {
		kvs = &FileSystemKVStore{basedir: basedir}
	}
	return
}

func (kvs *FileSystemKVStore) filename(key string) string {
	return filepath.Join(kvs.basedir, key)
}

// Get returns the specified key's value
func (kvs *FileSystemKVStore) Get(key string) ([]byte, error) {
	return lockedfile.Read(kvs.filename(key))
}

// Set sets the value of a specific key
func (kvs *FileSystemKVStore) Set(key string, value []byte) error {
	return lockedfile.Write(kvs.filename(key), bytes.NewReader(value), 0600)
}
