package fsx

import (
	"errors"
	"io/fs"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
)

// baseDir is the base directory we use for testing.
var baseDir = "./testdata/"

// failingStatFS is a fs.FS returning a file where stat() fails.
type failingStatFS struct {
	CloseCount *atomic.Int64
}

// failingStatFile is a fs.File where stat() fails.
type failingStatFile struct {
	CloseCount *atomic.Int64
}

// errStatFailed is the internal error indicating that stat() failed.
var errStatFailed = errors.New("stat failed")

// Stat is a stat implementation that fails.
func (failingStatFile) Stat() (os.FileInfo, error) {
	return nil, errStatFailed
}

// Open opens a fake file whose Stat fails.
func (f failingStatFS) Open(pathname string) (fs.File, error) {
	return failingStatFile(f), nil
}

// Close closes the failingStatFile.
func (fs failingStatFile) Close() error {
	if fs.CloseCount != nil {
		fs.CloseCount.Add(1)
	}
	return nil
}

// Read implements fs.File.Read.
func (failingStatFile) Read([]byte) (int, error) {
	return 0, errors.New("shouldn't be called")
}

func TestOpenWithFailingStat(t *testing.T) {
	count := &atomic.Int64{}
	_, err := openWithFS(
		failingStatFS{CloseCount: count}, baseDir+"testfile.txt")
	if !errors.Is(err, errStatFailed) {
		t.Error("expected error with invalid FS", err)
	}
	if count.Load() != 1 {
		t.Error("expected close counter to be equal to 1")
	}
}

func TestOpenNonexistentFile(t *testing.T) {
	_, err := OpenFile(baseDir + "invalidtestfile.txt")
	if !errors.Is(err, syscall.ENOENT) {
		t.Errorf("not the error we expected")
	}
}

func TestOpenDirectoryShouldFail(t *testing.T) {
	_, err := OpenFile(baseDir)
	if !errors.Is(err, syscall.EISDIR) {
		t.Fatalf("not the error we expected: %+v", err)
	}
}

func TestOpeningExistingFileShouldWork(t *testing.T) {
	file, err := OpenFile(baseDir + "testfile.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
}
