package sessionresolver

import (
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestErrWrapper(t *testing.T) {
	ew := &errwrapper{
		error: io.EOF,
		URL:   "https://dns.quad9.net/dns-query",
	}
	o := ew.Error()
	expect := "<https://dns.quad9.net/dns-query> EOF"
	if diff := cmp.Diff(expect, o); diff != "" {
		t.Fatal(diff)
	}
}
