package enginenetx

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestStreamTacticsFromSlice(t *testing.T) {
	input := []*httpsDialerTactic{}
	ff := &testingx.FakeFiller{}
	ff.Fill(&input)

	var output []*httpsDialerTactic
	for tx := range streamTacticsFromSlice(input) {
		output = append(output, tx)
	}

	if diff := cmp.Diff(input, output); diff != "" {
		t.Fatal(diff)
	}
}
