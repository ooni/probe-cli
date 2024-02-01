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

// golangCorrectVersionCheckP returns whether we're using the correct golang version.
func golangCorrectVersionCheckP(filename string) bool {
	expected := string(must.FirstLineBytes(must.ReadFile(filename)))

	// read the version of go that we're using
	firstline := string(must.FirstLineBytes(must.RunOutput(log.Log, "go", "version")))
	vec := strings.Split(firstline, " ")
	runtimex.Assert(len(vec) == 4, "expected four tokens")
	if got := vec[2]; got != "go"+expected {
		log.Warnf("expected go%s but got %s", expected, got)
		return false
	}
	log.Infof("using go%s", expected)
	return true
}

// golangCheck checks whether the "go" binary is the correct version
func golangCheck(filename string) {
	runtimex.Assert(golangCorrectVersionCheckP(filename), "invalid Go version")
}

// golangGOPATH returns the GOPATH value.
func golangGOPATH() string {
	return string(must.FirstLineBytes(must.RunOutput(log.Log, "go", "env", "GOPATH")))
}
