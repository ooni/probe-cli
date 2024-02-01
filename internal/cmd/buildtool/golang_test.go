package main

import (
	"os"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestGolangBinary(t *testing.T) {
	// make sure the code does not panic when it runs and returns a valid binary
	oldDirectory := runtimex.Try1(os.Getwd())
	runtimex.Try0(os.Chdir("../../.."))
	value := golangBinary()
	must.RunQuiet(value, "version")
	runtimex.Try0(os.Chdir(oldDirectory))
}
