package main

import (
	"os"
	"sync"
)

// MkdirTemp creates a new temporary directory.
func MkdirTemp(dir string, pattern string) *TempDir {
	name, err := os.MkdirTemp(dir, pattern)
	FatalOnError(err, "cannot create temporary directory")
	return &TempDir{name: name}
}

// TempDir is a temporary directory.
type TempDir struct {
	name string
}

// Path returns the directory name.
func (d *TempDir) Path() string {
	return d.name
}

// Cleanup removes the temporary directory.
func (d *TempDir) Cleanup() {
	os.RemoveAll(d.name)
}

// File is a wrapper for *os.File.
type File struct {
	f    *os.File
	once *sync.Once
}

// CreateFileTemp creates a new temporary file.
func CreateFileTemp(dir string, pattern string) *File {
	f, err := os.CreateTemp(dir, pattern)
	FatalOnError(err, "cannot create temporary file")
	return &File{f: f, once: &sync.Once{}}
}

// CreateFile creates a new file.
func CreateFile(fname string) *File {
	f, err := os.Create(fname)
	FatalOnError(err, "cannot create file")
	return &File{f: f, once: &sync.Once{}}
}

// Name returns the file name.
func (f *File) Name() string {
	return f.f.Name()
}

// WriteString writes a string into the file.
func (f *File) WriteString(s string) {
	_, err := f.f.WriteString(s)
	FatalOnError(err, "cannot write into file")
}

// MustClose closes the file and exit in case of error.
func (f *File) MustClose() {
	f.once.Do(func() {
		err := f.f.Close()
		FatalOnError(err, "cannot close file")
	})
}
