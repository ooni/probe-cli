package kvstore

import (
	"errors"
	"testing"
)

func TestNoSuchKey(t *testing.T) {
	kvs := &Memory{}
	value, err := kvs.Get("nonexistent")
	if !errors.Is(err, ErrNoSuchKey) {
		t.Fatal("expected an error here")
	}
	if value != nil {
		t.Fatal("expected empty string here")
	}
}

func TestExistingKey(t *testing.T) {
	kvs := &Memory{}
	if err := kvs.Set("antani", []byte("mascetti")); err != nil {
		t.Fatal(err)
	}
	value, err := kvs.Get("antani")
	if err != nil {
		t.Fatal(err)
	}
	if string(value) != "mascetti" {
		t.Fatal("not the result we expected")
	}
}
