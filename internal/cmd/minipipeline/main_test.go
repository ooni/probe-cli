package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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

func TestMainSuccess(t *testing.T) {
	// make sure we set the destination directory
	*destdir = "xo"

	// make sure we're reading from the expected input
	*measurement = filepath.Join("testdata", "measurement.json")

	// make sure we store the expected output
	contentmap := make(map[string][]byte)
	mustWriteFileFn = func(filename string, content []byte, mode fs.FileMode) {
		contentmap[filename] = content
	}

	// make sure osExit is correct
	osExit = os.Exit

	// run the main function
	main()

	// make sure the generated observations are good
	expectedObservations := mustloadfile(filepath.Join("testdata", "observations.json"))
	gotObservations := mustloaddata(contentmap, filepath.Join("xo", "observations.json"))
	if diff := cmp.Diff(expectedObservations, gotObservations); diff != "" {
		t.Fatal(diff)
	}

	// make sure the generated analysis is good
	expectedAnalysis := mustloadfile(filepath.Join("testdata", "analysis.json"))
	gotAnalysis := mustloaddata(contentmap, filepath.Join("xo", "analysis.json"))
	if diff := cmp.Diff(expectedAnalysis, gotAnalysis); diff != "" {
		t.Fatal(diff)
	}
}

func TestMainUsage(t *testing.T) {
	// make sure we clear the destination directory
	*destdir = ""

	// make sure the expected input file is empty
	*measurement = ""

	// make sure we panic if we try to write on disk
	mustWriteFileFn = func(filename string, content []byte, mode fs.FileMode) {
		panic(errors.New("mustWriteFileFn"))
	}

	// make sure osExit is correct
	osExit = func(code int) {
		panic(fmt.Errorf("osExit: %d", code))
	}

	var err error
	func() {
		// intercept panic caused by osExit or other panics
		defer func() {
			if r := recover(); r != nil {
				err = r.(error)
			}
		}()

		// run the main function with the given args
		main()
	}()

	// make sure we've got the expected error
	if err == nil || err.Error() != "osExit: 1" {
		t.Fatal("expected", "os.Exit: 1", "got", err)
	}
}
