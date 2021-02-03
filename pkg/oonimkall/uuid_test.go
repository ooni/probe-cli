package oonimkall_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/pkg/oonimkall"
)

func TestNewUUID4(t *testing.T) {
	if out := oonimkall.NewUUID4(); len(out) != 36 {
		t.Fatal("not the expected output")
	}
}
