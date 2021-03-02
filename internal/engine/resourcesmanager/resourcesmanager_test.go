package resourcesmanager

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
)

func TestAllGood(t *testing.T) {
	// make sure we start from scratch
	if err := os.RemoveAll("testdata"); err != nil {
		t.Fatal(err)
	}
	// first iteration should copy the resources
	cw := &CopyWorker{DestDir: "testdata"}
	if err := cw.Ensure(); err != nil {
		t.Fatal(err)
	}
	// second iteration should just ensure they're there
	if err := cw.Ensure(); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyDestDir(t *testing.T) {
	cw := &CopyWorker{DestDir: ""}
	if err := cw.Ensure(); !errors.Is(err, ErrDestDirEmpty) {
		t.Fatal("not the error we expected", err)
	}
}

func TestMkdirAllFailure(t *testing.T) {
	errMocked := errors.New("mocked error")
	cw := &CopyWorker{
		DestDir: "testdata",
		MkdirAll: func(path string, perm os.FileMode) error {
			return errMocked
		},
	}
	if err := cw.Ensure(); !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
}

func TestOpenFailure(t *testing.T) {
	errMocked := errors.New("mocked error")
	cw := &CopyWorker{
		DestDir: "testdata",
		MkdirAll: func(path string, perm os.FileMode) error {
			return nil
		},
		ReadFile: func(path string) ([]byte, error) {
			return []byte(`fake`), nil
		},
		Equal: func(left, right string) bool {
			return false
		},
		Open: func(path string) (fs.File, error) {
			return nil, errMocked
		},
	}
	if err := cw.Ensure(); !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
}

func TestNewReaderFailure(t *testing.T) {
	errMocked := errors.New("mocked error")
	cw := &CopyWorker{
		DestDir: "testdata",
		MkdirAll: func(path string, perm os.FileMode) error {
			return nil
		},
		Equal: func(left, right string) bool {
			return false
		},
		NewReader: func(r io.Reader) (io.ReadCloser, error) {
			return nil, errMocked
		},
	}
	if err := cw.Ensure(); !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
}

func TestReadAllFailure(t *testing.T) {
	errMocked := errors.New("mocked error")
	cw := &CopyWorker{
		DestDir: "testdata",
		MkdirAll: func(path string, perm os.FileMode) error {
			return nil
		},
		Equal: func(left, right string) bool {
			return false
		},
		ReadAll: func(r io.Reader) ([]byte, error) {
			return nil, errMocked
		},
	}
	if err := cw.Ensure(); !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
}

func TestSHA256Mismatch(t *testing.T) {
	cw := &CopyWorker{
		DestDir: "testdata",
		MkdirAll: func(path string, perm os.FileMode) error {
			return nil
		},
		Equal: func(left, right string) bool {
			return false
		},
		Different: func(left, right string) bool {
			return true
		},
	}
	if err := cw.Ensure(); !errors.Is(err, ErrSHA256Mismatch) {
		t.Fatal("not the error we expected", err)
	}
}

func TestWriteFileFailure(t *testing.T) {
	errMocked := errors.New("mocked error")
	cw := &CopyWorker{
		DestDir: "testdata",
		MkdirAll: func(path string, perm os.FileMode) error {
			return nil
		},
		Equal: func(left, right string) bool {
			return false
		},
		WriteFile: func(filename string, data []byte, perm fs.FileMode) error {
			return errMocked
		},
	}
	if err := cw.Ensure(); !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
}
