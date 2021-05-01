package kvstore

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rogpeppe/go-internal/lockedfile"
)

// FS is a file-system based KVStore.
type FS struct {
	basedir string
}

// NewFS creates a new kvstore.FileSystem.
func NewFS(basedir string) (kvs *FS, err error) {
	return newFileSystem(basedir, os.MkdirAll)
}

// osMkdirAll is the type of os.MkdirAll.
type osMkdirAll func(path string, perm fs.FileMode) error

// newFileSystem is like NewFileSystem with a customizable
// osMkdirAll function for creating the kvstore dir.
func newFileSystem(basedir string, mkdir osMkdirAll) (*FS, error) {
	if err := mkdir(basedir, 0700); err != nil {
		return nil, err
	}
	return &FS{basedir: basedir}, nil
}

// filename returns the filename for a given key.
func (kvs *FS) filename(key string) string {
	return filepath.Join(kvs.basedir, key)
}

// Get returns the specified key's value. In case of error, the
// error type is such that errors.Is(err, ErrNoSuchKey).
func (kvs *FS) Get(key string) ([]byte, error) {
	data, err := lockedfile.Read(kvs.filename(key))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNoSuchKey, err.Error())
	}
	return data, nil
}

// Set sets the value of a specific key.
func (kvs *FS) Set(key string, value []byte) error {
	return lockedfile.Write(kvs.filename(key), bytes.NewReader(value), 0600)
}
