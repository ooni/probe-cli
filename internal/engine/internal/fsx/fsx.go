// Package fsx contains file system extension
package fsx

import (
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// Open is a wrapper for os.Open that ensures that we're opening a file.
func Open(pathname string) (fs.File, error) {
	return OpenWithFS(filesystem{}, pathname)
}

// OpenWithFS is like Open but with explicit file system argument.
func OpenWithFS(fs fs.FS, pathname string) (fs.File, error) {
	file, err := fs.Open(pathname)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if info.IsDir() {
		file.Close()
		return nil, fmt.Errorf(
			"input path points to a directory: %w", syscall.EISDIR)
	}
	return file, nil
}

// filesystem is a private implementation of fs.FS.
type filesystem struct{}

// Open implements fs.FS.Open.
func (filesystem) Open(pathname string) (fs.File, error) {
	return os.Open(pathname)
}
