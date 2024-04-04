package main

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/version"
)

func TestSimple(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	MainWithConfiguration("example", &Options{
		SoftwareName:    "miniooni",
		SoftwareVersion: version.Version,
		Yes:             true,
	})
}
