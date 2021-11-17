// Package fsx contains io/fs extensions.
package fsx

import (
	"io/fs"
	"os"
	"syscall"
)

// OpenFile is a wrapper for os.OpenFile that ensures that
// we're opening a file rather than a directory. If you are
// opening a directory, this func returns an *os.PathError
// error with Err set to syscall.EISDIR.
//
// As mentioned in CONTRIBUTING.md, this is the function
// you SHOULD be using when opening files.
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
		return nil, &os.PathError{
			Op:   "openFile",
			Path: pathname,
			Err:  syscall.EISDIR,
		}
	}
	return file, nil
}

// filesystem is a private implementation of fs.FS.
type filesystem struct{}

// Open implements fs.FS.Open.
func (filesystem) Open(pathname string) (fs.File, error) {
	return os.Open(pathname)
}
