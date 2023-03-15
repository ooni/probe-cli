// Package fsx contains io/fs extensions.
package fsx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// OpenFile is a wrapper for os.OpenFile that ensures that
// we're opening a file rather than a directory. If you are
// not opening a regular file, this func returns an error.
//
// As mentioned in CONTRIBUTING.md, this is the function
// you SHOULD be using when opening files.
func OpenFile(pathname string) (fs.File, error) {
	return openWithFS(filesystem{}, pathname)
}

// ErrNotRegularFile indicates you're not opening a regular file.
var ErrNotRegularFile = errors.New("not a regular file")

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
	if !isRegular(info) {
		file.Close()
		return nil, fmt.Errorf("%w: %s", ErrNotRegularFile, pathname)
	}
	return file, nil
}

// filesystem is a private implementation of fs.FS.
type filesystem struct{}

// Open implements fs.FS.Open.
func (filesystem) Open(pathname string) (fs.File, error) {
	return os.Open(pathname)
}

func isRegular(info fs.FileInfo) bool {
	return info.Mode().IsRegular()
}

// RegularFileExists returns whether the given filename
// exists and is a regular file.
func RegularFileExists(filename string) bool {
	finfo, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return isRegular(finfo)
}

// DirectoryExists returns whether the given filename
// exists and is a directory.
func DirectoryExists(filename string) bool {
	finfo, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return finfo.IsDir()
}
