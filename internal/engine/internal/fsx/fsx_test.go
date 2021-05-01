package fsx_test

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/fsx"
)

var StateBaseDir = "./testdata/"

type FailingStatFS struct {
	CloseCount *atomicx.Int64
}

type FailingStatFile struct {
	CloseCount *atomicx.Int64
}

var errStatFailed = errors.New("stat failed")

func (FailingStatFile) Stat() (os.FileInfo, error) {
	return nil, errStatFailed
}

func (f FailingStatFS) Open(pathname string) (fs.File, error) {
	return FailingStatFile(f), nil
}

func (fs FailingStatFile) Close() error {
	if fs.CloseCount != nil {
		fs.CloseCount.Add(1)
	}
	return nil
}

func (FailingStatFile) Read([]byte) (int, error) {
	return 0, nil
}

func TestOpenWithFailingStat(t *testing.T) {
	count := &atomicx.Int64{}
	_, err := fsx.OpenWithFS(FailingStatFS{CloseCount: count}, StateBaseDir+"testfile.txt")
	if !errors.Is(err, errStatFailed) {
		t.Errorf("expected error with invalid FS: %+v", err)
	}
	if count.Load() != 1 {
		t.Error("expected counter to be equal to 1")
	}
}

func TestOpenNonexistentFile(t *testing.T) {
	_, err := fsx.Open(StateBaseDir + "invalidtestfile.txt")
	if !errors.Is(err, syscall.ENOENT) {
		t.Errorf("not the error we expected")
	}
}

func TestOpenDirectoryShouldFail(t *testing.T) {
	_, err := fsx.Open(StateBaseDir)
	if !errors.Is(err, syscall.EISDIR) {
		t.Fatalf("not the error we expected: %+v", err)
	}
}

func TestOpeningExistingFileShouldWork(t *testing.T) {
	file, err := fsx.Open(StateBaseDir + "testfile.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
}
