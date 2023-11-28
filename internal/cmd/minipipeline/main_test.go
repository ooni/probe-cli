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
	// reconfigure the global options for main
	*destdirFlag = "xo"
	*measurementFlag = filepath.Join("testdata", "measurement.json")
	contentmap := make(map[string][]byte)
	mustWriteFileFn = func(filename string, content []byte, mode fs.FileMode) {
		contentmap[filename] = content
	}
	osExitFn = os.Exit
	*prefixFlag = "y-"

	// run the main function
	main()

	// make sure the generated observations are good
	expectedObservations := mustloadfile(filepath.Join("testdata", "observations.json"))
	gotObservations := mustloaddata(contentmap, filepath.Join("xo", "y-observations.json"))
	if diff := cmp.Diff(expectedObservations, gotObservations); diff != "" {
		t.Fatal(diff)
	}

	// make sure the generated analysis is good
	expectedAnalysis := mustloadfile(filepath.Join("testdata", "analysis.json"))
	gotAnalysis := mustloaddata(contentmap, filepath.Join("xo", "y-analysis.json"))
	if diff := cmp.Diff(expectedAnalysis, gotAnalysis); diff != "" {
		t.Fatal(diff)
	}
}

func TestMainUsage(t *testing.T) {
	// reconfigure the global options for main
	*destdirFlag = ""
	*measurementFlag = ""
	mustWriteFileFn = func(filename string, content []byte, mode fs.FileMode) {
		panic(errors.New("mustWriteFileFn"))
	}
	osExitFn = func(code int) {
		panic(fmt.Errorf("osExit: %d", code))
	}
	*prefixFlag = ""

	// run the main function
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
