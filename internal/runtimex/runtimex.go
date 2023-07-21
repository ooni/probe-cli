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

// setkv is a convenience function to set a [BuildInfoRecord] entry.
func (bir *BuildInfoRecord) setkv(key, value string) {
	switch key {
	case "vcs.revision":
		bir.VcsRevision = value
	case "vcs.time":
		bir.VcsTime = value
	case "vcs.modified":
		bir.VcsModified = value
	case "vcs":
		bir.VcsTool = value
	}
}

// setall sets all the possible settings.
func (bir *BuildInfoRecord) setall(settings []debug.BuildSetting) {
	for _, entry := range settings {
		bir.setkv(entry.Key, entry.Value)
	}
}

// BuildInfo is the singleton containing build-time information.
var BuildInfo = &BuildInfoRecord{}

func init() {
	info, good := debug.ReadBuildInfo()
	if good {
		BuildInfo.GoVersion = info.GoVersion
		BuildInfo.setall(info.Settings)
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

// Try0 calls [runtimex.PanicOnError] if err is not nil.
func Try0(err error) {
	PanicOnError(err, "Try0")
}

// Try1 is like [Try0] but supports functions returning one values and an error.
func Try1[T1 any](v1 T1, err error) T1 {
	PanicOnError(err, "Try1")
	return v1
}

// Try2 is like [Try1] but supports functions returning two values and an error.
func Try2[T1, T2 any](v1 T1, v2 T2, err error) (T1, T2) {
	PanicOnError(err, "Try2")
	return v1, v2
}

// Try3 is like [Try2] but supports functions returning three values and an error.
func Try3[T1, T2, T3 any](v1 T1, v2 T2, v3 T3, err error) (T1, T2, T3) {
	PanicOnError(err, "Try3")
	return v1, v2, v3
}

// WarningLogger is a logger that emits formatted warnings. We cannot directly use the definition
// inside the [model] package because [model] depends on [runtimex].
type WarningLogger interface {
	Warnf(format string, v ...any)
}

// CatchLogAndIgnorePanic is a function that catches and ignores panics. You
// can invoke this function as follows:
//
//	defer runtimex.CatchLogAndIgnorePanic(logger, "prefix.......")
//
// and rest assured that any panic will not propagate further. You should typically
// only use this function when writing testing code.
//
// This function will emit a warning message prefixed using the given prefix and emitted
// using the given logger in case it intercepts and suppresses a panic.
func CatchLogAndIgnorePanic(logger WarningLogger, prefix string) {
	if rec := recover(); rec != nil {
		logger.Warnf("%s: %+v", prefix, rec)
	}
}
