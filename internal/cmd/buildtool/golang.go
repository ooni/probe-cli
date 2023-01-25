package main

//
// Getting the correct version of Go.
//

import (
	"strings"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// golangCheck checks whether the "go" binary is the correct version
func golangCheck(filename string) {
	expected := string(must.FirstLineBytes(must.ReadFile(filename)))
	firstline := string(must.FirstLineBytes(must.RunOutput(log.Log, "go", "version")))
	vec := strings.Split(firstline, " ")
	runtimex.Assert(len(vec) == 4, "expected four tokens")
	if got := vec[2]; got != "go"+expected {
		log.Fatalf("expected go%s but got %s", expected, got)
	}
	log.Infof("using go%s", expected)
}
