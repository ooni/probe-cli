// Package fsx contains io/fs extensions.
package fsx

import (
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// OpenFile is a wrapper for os.OpenFile that ensures that
// we're opening a file rather than a directory. If you are
// opening a directory, this func will return an error.
func OpenFile(pathname string) (fs.File, error) {
	return openWithFS(filesystem{}, pathname)
}

// openWithFS is like Open but with explicit file system argument.
func openWithFS(fs fs.FS, pathname string) (fs.File, error) {
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
