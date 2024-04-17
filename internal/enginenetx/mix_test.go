package enginenetx

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestMixSequentially(t *testing.T) {
	primary := []*httpsDialerTactic{}
	fallback := []*httpsDialerTactic{}

	ff := &testingx.FakeFiller{}
	ff.Fill(&primary)
	ff.Fill(&fallback)

	expect := append([]*httpsDialerTactic{}, primary...)
	expect = append(expect, fallback...)

	var output []*httpsDialerTactic
	for tx := range mixSequentially(streamTacticsFromSlice(primary), streamTacticsFromSlice(fallback)) {
		output = append(output, tx)
	}

	if diff := cmp.Diff(expect, output); diff != "" {
		t.Fatal(diff)
	}
}
