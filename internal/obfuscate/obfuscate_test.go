package obfuscate_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/obfuscate"
)

func TestRoundTrip(t *testing.T) {
	input := []byte("The quick brown fox jumps over the lazy dog")
	t.Log(input)

	intermediate := obfuscate.Apply(input)
	t.Log(intermediate)

	if diff := cmp.Diff(input, intermediate); diff == "" {
		t.Fatal("intermediate and input should be different")
	}

	output := obfuscate.Apply(intermediate)
	t.Log(output)

	if diff := cmp.Diff(input, output); diff != "" {
		t.Fatal(diff)
	}
}
