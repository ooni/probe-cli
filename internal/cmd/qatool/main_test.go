package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestMainList(t *testing.T) {
	// reconfigure the global options for main
	*destdirFlag = ""
	*listFlag = true
	mustReadFileFn = func(filename string) []byte {
		panic(errors.New("mustReadFileFn"))
	}
	mustWriteFileFn = func(filename string, content []byte, mode fs.FileMode) {
		panic(errors.New("mustWriteFileFn"))
	}
	osExitFn = func(code int) {
		panic(fmt.Errorf("osExit: %d", code))
	}
	osMkdirAllFn = func(path string, perm os.FileMode) error {
		panic(errors.New("osMkdirAllFn"))
	}
	*runFlag = ""

	// run the main function
	main()
}

func TestMainSuccess(t *testing.T) {
	// reconfigure the global options for main
	*destdirFlag = "xo"
	*listFlag = false
	contentmap := make(map[string][]byte)
	mustReadFileFn = func(filename string) []byte {
		data, found := contentmap[filename]
		runtimex.Assert(found, fmt.Sprintf("cannot find %s", filename))
		return data
	}
	mustWriteFileFn = func(filename string, content []byte, mode fs.FileMode) {
		// make sure we can parse as JSON
		var container map[string]any
		if err := json.Unmarshal(content, &container); err != nil {
			t.Fatal(err)
		}

		// register we have written a file
		contentmap[filename] = content
	}
	osExitFn = os.Exit
	osMkdirAllFn = func(path string, perm os.FileMode) error {
		return nil
	}
	*runFlag = "dnsBlocking"

	// run the main function
	main()

	// make sure we attempted to write the desired files
	expect := map[string]bool{
		"xo/dnsBlockingBOGON/measurement.json":                  true,
		"xo/dnsBlockingBOGON/observations.json":                 true,
		"xo/dnsBlockingBOGON/analysis.json":                     true,
		"xo/dnsBlockingNXDOMAIN/measurement.json":               true,
		"xo/dnsBlockingNXDOMAIN/observations.json":              true,
		"xo/dnsBlockingNXDOMAIN/analysis.json":                  true,
		"xo/dnsBlockingAndroidDNSCacheNoData/measurement.json":  true,
		"xo/dnsBlockingAndroidDNSCacheNoData/observations.json": true,
		"xo/dnsBlockingAndroidDNSCacheNoData/analysis.json":     true,
	}
	got := make(map[string]bool)
	for key := range contentmap {
		got[key] = true
	}
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Fatal(diff)
	}
}

func TestMainUsage(t *testing.T) {
	// reconfigure the global options for main
	*destdirFlag = ""
	*listFlag = false
	mustReadFileFn = func(filename string) []byte {
		panic(errors.New("mustReadFileFn"))
	}
	mustWriteFileFn = func(filename string, content []byte, mode fs.FileMode) {
		panic(errors.New("mustWriteFileFn"))
	}
	osExitFn = func(code int) {
		panic(fmt.Errorf("osExit: %d", code))
	}
	osMkdirAllFn = func(path string, perm os.FileMode) error {
		panic(errors.New("osMkdirAllFn"))
	}
	*runFlag = ""

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
