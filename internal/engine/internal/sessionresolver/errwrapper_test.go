package sessionresolver

import (
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestErrWrapper(t *testing.T) {
	ew := newErrWrapper(io.EOF, "https://dns.quad9.net/dns-query")
	o := ew.Error()
	expect := "<https://dns.quad9.net/dns-query> EOF"
	if diff := cmp.Diff(expect, o); diff != "" {
		t.Fatal(diff)
	}
	if !errors.Is(ew, io.EOF) {
		t.Fatal("not the sub-error we expected")
	}
	if errors.Unwrap(ew) != io.EOF {
		t.Fatal("unwrap failed")
	}
}
