package main

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

// cBuildExportEnviron merges the gloval and the local c build environment
// to produce the environment variables to export for the build. More specifically,
// this appends the local variables to the remote variables for any string slice
// type inside [cBuildEnv]. We ignore all the other variables.
func cBuildExportEnviron(global, local *cBuildEnv) *shellx.Envp {
	envp := &shellx.Envp{}

	cflags := append([]string{}, global.cflags...)
	cflags = append(cflags, local.cflags...)
	envp.Append("CFLAGS", strings.Join(cflags, " "))

	cxxflags := append([]string{}, global.cxxflags...)
	cxxflags = append(cxxflags, local.cxxflags...)
	envp.Append("CXXFLAGS", strings.Join(cxxflags, " "))

	ldflags := append([]string{}, global.ldflags...)
	ldflags = append(ldflags, local.ldflags...)
	envp.Append("LDFLAGS", strings.Join(ldflags, " "))

	return envp
}
