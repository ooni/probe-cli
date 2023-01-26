package main

//
// Common build environment for builds using C (which applies
// to both CGO builds and to pure C builds).
//

import (
	"strings"

	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// cBuildEnv describes the C build environment. We use
// this structure for more complex C and CGO builds.
type cBuildEnv struct {
	// binpath is the path containing the C and C++ compilers.
	binpath string

	// cc is the full path to the C compiler.
	cc string

	// cflags contains the extra CFLAGS to set.
	cflags []string

	// configureHost is the value to pass to ./configure's --host option.
	configureHost string

	// destdir is the directory where to install.
	destdir string

	// cxx is the full path to the CXX compiler.
	cxx string

	// cxxflags contains the extra CXXFLAGS to set.
	cxxflags []string

	// goarch is the GOARCH we're building for.
	goarch string

	// goarm is the GOARM subarchitecture.
	goarm string

	// lfdlags contains the LDFLAGS to use when compiling.
	ldflags []string

	// openSSLAPIDefine is an extra define we need to add on Android.
	openSSLAPIDefine string

	// openSSLCompiler is the compiler name for OpenSSL.
	openSSLCompiler string
}

// cBuildExportEnviron merges the global and the local [cBuildEnv] to produce
// environment variables suitable for cross compiling. More specifically:
//
// 1. we use the CC, CXX, LD, etc. scalars from the global environment.
//
// 2. we append all the vector variables defined in the local environment
// to the ones inside the global environment.
//
// In other words, the local environment is only suitable for appending
// new values to CFLAGS, CXXFLAGS, LDFLAGS, etc.
func cBuildExportEnviron(global, local *cBuildEnv) *shellx.Envp {
	envp := &shellx.Envp{}

	if global.cc != "" {
		envp.Append("CC", global.cc)
	}
	if global.cxx != "" {
		envp.Append("CXX", global.cxx)
	}

	cflags := append([]string{}, global.cflags...)
	cflags = append(cflags, local.cflags...)
	if len(cflags) > 0 {
		envp.Append("CFLAGS", strings.Join(cflags, " "))
	}

	cxxflags := append([]string{}, global.cxxflags...)
	cxxflags = append(cxxflags, local.cxxflags...)
	if len(cxxflags) > 0 {
		envp.Append("CXXFLAGS", strings.Join(cxxflags, " "))
	}

	ldflags := append([]string{}, global.ldflags...)
	ldflags = append(ldflags, local.ldflags...)
	if len(ldflags) > 0 {
		envp.Append("LDFLAGS", strings.Join(ldflags, " "))
	}

	return envp
}
