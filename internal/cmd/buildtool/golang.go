package main

//
// Getting the correct version of Go.
//

import (
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// golangCorrectVersionCheckP returns whether we're using the correct golang version.
func golangCorrectVersionCheckP(filename string) bool {
	expected := string(must.FirstLineBytes(must.ReadFile(filename)))
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

// golangOsExit allows to test that [golangCheck] invokes [os.Exit] with exit code 1
// whenever the version of golang is not the intended one.
var golangOsExit = os.Exit

// golangCheck checks whether the "go" binary is the correct version
func golangCheck(filename string) {
	if !golangCorrectVersionCheckP(filename) {
		golangOsExit(1)
	}
}

// golangGOPATH returns the GOPATH value.
func golangGOPATH() string {
	return string(must.FirstLineBytes(must.RunOutput(log.Log, "go", "env", "GOPATH")))
}
