package ooapi

import "testing"

func TestNewQueryFieldBoolWorks(t *testing.T) {
	if s := newQueryFieldBool(true); s != "true" {
		t.Fatal("invalid encoding of true")
	}
	if s := newQueryFieldBool(false); s != "false" {
		t.Fatal("invalid encoding of false")
	}
}
