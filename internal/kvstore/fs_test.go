package kvstore

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSystemGood(t *testing.T) {
	dirpath := filepath.Join("testdata", "kvstore2")
	if err := os.RemoveAll(dirpath); err != nil {
		t.Fatal(err)
	}
	kvstore, err := NewFS(dirpath)
	if err != nil {
		t.Fatal(err)
	}
	value := []byte("foobar")
	if err := kvstore.Set("antani", value); err != nil {
		t.Fatal(err)
	}
	ovalue, err := kvstore.Get("antani")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ovalue, value) {
		t.Fatal("invalid value")
	}
}

func TestFileSystemNoSuchKey(t *testing.T) {
	dirpath := filepath.Join("testdata", "kvstore2")
	if err := os.RemoveAll(dirpath); err != nil {
		t.Fatal(err)
	}
	kvstore, err := NewFS(dirpath)
	if err != nil {
		t.Fatal(err)
	}
	value, err := kvstore.Get("antani")
	if !errors.Is(err, ErrNoSuchKey) {
		t.Fatal("not the error we expected", err)
	}
	if value != nil {
		t.Fatal("expected nil value")
	}
}

func TestFileSystemWithFailure(t *testing.T) {
	expect := errors.New("mocked error")
	mkdir := func(path string, perm fs.FileMode) error {
		return expect
	}
	kvstore, err := newFileSystem(
		filepath.Join("testdata", "kvstore2"),
		mkdir,
	)
	if !errors.Is(err, expect) {
		t.Fatal("not the error we expected", err)
	}
	if kvstore != nil {
		t.Fatal("expected nil here")
	}
}
