package main

//
// Getting the correct version of Go.
//

import (
	"encoding/json"
	"go/version"
	"strings"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// goModInfo
type goModInfo struct {
	// Go is the `go` directive
	Go string

	// Toolchain is the `toolchain` directive.
	Toolchain string
}

// golangReadGoMod returns the relevant directives from the module's go.mod,
// which is the single source of truth for the Go version we pin.
func golangReadGoMod() *goModInfo {
	var info goModInfo
	runtimex.Try0(json.Unmarshal(
		must.RunOutput(log.Log, "go", "mod", "edit", "-json"), &info))
	return &info
}

// golangRunningVersion returns the version of the "go" binary on the PATH
// as a canonical Go version string (e.g. "go1.27.3").
func golangRunningVersion() string {
	firstline := string(must.FirstLineBytes(must.RunOutput(log.Log, "go", "version")))
	vec := strings.Split(firstline, " ")
	runtimex.Assert(len(vec) == 4, "expected four tokens")
	return vec[2]
}

// golangVersionCheckP reports whether the running Go version satisfies the
// minimum required by go.mod's `go` directive. We only enforce a floor.
func golangVersionCheckP(minimum, running string) bool {
	want := "go" + minimum
	if version.Compare(running, want) < 0 {
		log.Warnf("expected go >= %s but got %s", want, running)
		return false
	}
	log.Infof("using %s (>= %s)", running, want)
	return true
}

// golangCheck ensures the "go" binary is new enough to build this module.
func golangCheck() {
	info := golangReadGoMod()
	runtimex.Assert(golangVersionCheckP(info.Go, golangRunningVersion()), "invalid Go version")
}

// golangToolchainVersion returns the exact Go toolchain version pinned by
// go.mod, falling back to the `go` directive when there is no `toolchain` line.
func golangToolchainVersion() string {
	info := golangReadGoMod()
	if info.Toolchain != "" {
		return strings.TrimPrefix(info.Toolchain, "go")
	}
	return info.Go
}

// golangGOPATH returns the GOPATH value.
func golangGOPATH() string {
	return string(must.FirstLineBytes(must.RunOutput(log.Log, "go", "env", "GOPATH")))
}
