package main

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/must"
)

func TestGolangBinary(t *testing.T) {
	// make sure the code does not panic when it runs and returns a valid binary
	value := golangBinary()
	must.RunQuiet(value, "version")
}
