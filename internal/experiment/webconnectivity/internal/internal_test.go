package internal_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity/internal"
)

func TestStringPointerToString(t *testing.T) {
	s := "ANTANI"
	if internal.StringPointerToString(&s) != s {
		t.Fatal("unexpected result")
	}
	if internal.StringPointerToString(nil) != "nil" {
		t.Fatal("unexpected result")
	}
}

func TestBoolPointerToString(t *testing.T) {
	v := true
	if internal.BoolPointerToString(&v) != "true" {
		t.Fatal("unexpected result")
	}
	v = false
	if internal.BoolPointerToString(&v) != "false" {
		t.Fatal("unexpected result")
	}
	if internal.BoolPointerToString(nil) != "nil" {
		t.Fatal("unexpected result")
	}
}
