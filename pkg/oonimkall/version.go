package oonimkall

//
// Exposes the OONI measurement engine name and version to callers
// so they can record which engine and version they are embedding.
//

import "github.com/ooni/probe-cli/v3/internal/version"

// EngineName returns the name of the OONI measurement engine
func EngineName() string {
	return "ooniprobe-engine"
}

// EngineVersion returns the version of the OONI measurement engine
func EngineVersion() string {
	return version.Version
}
