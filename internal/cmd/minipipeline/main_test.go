package main

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/must"
)

func mustloadfile(filename string) (object map[string]any) {
	data := must.ReadFile(filename)
	must.UnmarshalJSON(data, &object)
	return
}

func mustloaddata(contentmap map[string][]byte, key string) (object map[string]any) {
	data := contentmap[key]
	must.UnmarshalJSON(data, &object)
	return
}

func TestMain(t *testing.T) {
	// make sure we're reading from the expected input
	inputs = []string{filepath.Join("testdata/measurement.json")}

	// make sure we store the expected output
	contentmap := make(map[string][]byte)
	mustWriteFileFn = func(filename string, content []byte, mode fs.FileMode) {
		contentmap[filename] = content
	}

	// run the main function
	main()

	// make sure the generated observations are good
	expectedObservations := mustloadfile(filepath.Join("testdata", "observations.json"))
	gotObservations := mustloaddata(contentmap, "observations-0000000000.json")
	if diff := cmp.Diff(expectedObservations, gotObservations); diff != "" {
		t.Fatal(diff)
	}

	// make sure the generated analysis is good
	expectedAnalysis := mustloadfile(filepath.Join("testdata", "analysis.json"))
	gotAnalysis := mustloaddata(contentmap, "analysis-0000000000.json")
	if diff := cmp.Diff(expectedAnalysis, gotAnalysis); diff != "" {
		t.Fatal(diff)
	}
}
