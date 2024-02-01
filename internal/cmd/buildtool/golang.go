package main

//
// Getting the correct version of Go.
//

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// golangCheckCorrectVersion returns true if the version of Go is correct.
func golangCheckCorrectVersion(filename string) bool {
	// read the version of go that we would like to use
	expected := string(must.FirstLineBytes(must.ReadFile(filename)))

	// read the version of go that we're using
	firstline := string(must.FirstLineBytes(must.RunOutput(log.Log, "go", "version")))
	vec := strings.Split(firstline, " ")
	runtimex.Assert(len(vec) == 4, "expected four tokens")

	// make sure they're equal
	return vec[2] == "go"+expected
}

// golangInstall installs and returns the path to the correct version of Go.
func golangInstall(filename string) string {
	// read the version of Go we would like to use
	expected := string(must.FirstLineBytes(must.ReadFile(filename)))

	// install the downloaded script
	packageName := fmt.Sprintf("golang.org/dl/go%s@latest", expected)
	must.Run(log.Log, "go", "install", "-v", packageName)

	// run the downloader script
	gobinary := filepath.Join(
		string(must.FirstLineBytes(must.RunOutput(log.Log, "go", "env", "GOPATH"))),
		"bin",
		fmt.Sprintf("go%s", expected),
	)
	must.Run(log.Log, gobinary, "download")

	// if all is good, then we have the right gobinary
	return gobinary
}

// golangBinaryWithoutCache returns the path to the correct golang binary to use.
func golangBinaryWithoutCache() string {
	if !golangCheckCorrectVersion("GOVERSION") {
		return golangInstall("GOVERSION")
	}
	return "go"
}

// golangCachedBinary is the cached golang binary.
var golangCachedBinary string

// golangCacheMu synchronizes accesses to [golangCachedBinary].
var golangCacheMu sync.Mutex

// golangBinary returns the path to the correct golang binary to use.
func golangBinary() string {
	defer golangCacheMu.Unlock()
	golangCacheMu.Lock()
	if golangCachedBinary == "" {
		golangCachedBinary = golangBinaryWithoutCache()
	}
	return golangCachedBinary
}

// golangGOPATH returns the GOPATH value.
func golangGOPATH() string {
	return string(must.FirstLineBytes(must.RunOutput(log.Log, "go", "env", "GOPATH")))
}
