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

// golangCheck checks whether the "go" binary is the correct version
func golangCheck() {
	expected := string(must.FirstLineBytes(must.ReadFile("GOVERSION")))
	firstline := string(must.FirstLineBytes(must.RunOutput(log.Log, "go", "version")))
	vec := strings.Split(firstline, " ")
	runtimex.Assert(len(vec) == 4, "expected four tokens")
	if got := vec[2]; got != "go"+expected {
		must.Fprintf(os.Stderr, "# FATAL: expected go%s but got %s", expected, got)
		os.Exit(1)
	}
	must.Fprintf(os.Stderr, "# using go%s\n", expected)
	must.Fprintf(os.Stderr, "\n")
}
