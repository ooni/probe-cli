package kvstore

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
)

func TestFileSystemGood(t *testing.T) {
	kvstore, err := NewFS(
		filepath.Join("testdata", "kvstore2"),
	)
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
