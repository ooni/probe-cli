// Package runtimex contains runtime extensions. This package is inspired to
// https://pkg.go.dev/github.com/m-lab/go/rtx, except that it's simpler.
package runtimex

import (
	"errors"
	"fmt"
	"runtime/debug"
)

// BuildInfoRecord contains build-time information.
type BuildInfoRecord struct {
	// GoVersion is the version of go with which this code
	// was compiled or an empty string.
	GoVersion string

	// VcsModified indicates whether the tree was dirty.
	VcsModified string

	// VcsRevision is the VCS revision we compiled.
	VcsRevision string

	// VcsTime is the time of the revision we're building.
	VcsTime string

	// VcsTool is the VCS tool being used.
	VcsTool string
}

// BuildInfo is the singleton containing build-time information.
var BuildInfo BuildInfoRecord

func init() {
	info, good := debug.ReadBuildInfo()
	if !good {
		return
	}
	BuildInfo.GoVersion = info.GoVersion
	for _, entry := range info.Settings {
		switch entry.Key {
		case "vcs.revision":
			BuildInfo.VcsRevision = entry.Value
		case "vcs.time":
			BuildInfo.VcsTime = entry.Value
		case "vcs.modified":
			BuildInfo.VcsModified = entry.Value
		case "vcs":
			BuildInfo.VcsTool = entry.Value
		}
	}
}

// PanicOnError calls panic() if err is not nil. The type passed
// to panic is an error type wrapping the original error.
func PanicOnError(err error, message string) {
	if err != nil {
		panic(fmt.Errorf("%s: %w", message, err))
	}
}

// Assert calls panic if assertion is false. The type passed to
// panic is an error constructed using errors.New(message).
func Assert(assertion bool, message string) {
	if !assertion {
		panic(errors.New(message))
	}
}

// PanicIfTrue calls panic if assertion is true. The type passed to
// panic is an error constructed using errors.New(message).
func PanicIfTrue(assertion bool, message string) {
	Assert(!assertion, message)
}

// PanicIfNil calls panic if the given interface is nil. The type passed to
// panic is an error constructed using errors.New(message).
func PanicIfNil(v any, message string) {
	PanicIfTrue(v == nil, message)
}
