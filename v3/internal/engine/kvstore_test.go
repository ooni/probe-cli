package engine

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestKVStoreIntegration(t *testing.T) {
	var (
		err     error
		kvstore KVStore
	)
	kvstore, err = NewFileSystemKVStore(
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
