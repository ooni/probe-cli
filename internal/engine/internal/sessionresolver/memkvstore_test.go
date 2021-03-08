package sessionresolver

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestKVStoreCustom(t *testing.T) {
	kvs := &memkvstore{}
	reso := &Resolver{KVStore: kvs}
	o := reso.kvstore()
	if o != kvs {
		t.Fatal("not the kvstore we expected")
	}
}

func TestMemkvstoreGetNotFound(t *testing.T) {
	reso := &Resolver{}
	key := "antani"
	out, err := reso.kvstore().Get(key)
	if !errors.Is(err, errMemkvstoreNotFound) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}

func TestMemkvstoreRoundTrip(t *testing.T) {
	reso := &Resolver{}
	key := []string{"antani", "mascetti"}
	value := [][]byte{[]byte(`mascetti`), []byte(`antani`)}
	for idx := 0; idx < 2; idx++ {
		if err := reso.kvstore().Set(key[idx], value[idx]); err != nil {
			t.Fatal(err)
		}
		out, err := reso.kvstore().Get(key[idx])
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(value[idx], out); diff != "" {
			t.Fatal(diff)
		}
	}
}
