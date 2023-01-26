package main

//
// Common build environment for builds using C (which applies
// to both CGO builds and to pure C builds).
//

import (
	"strings"

	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// cBuildEnv describes the C build environment. We use this structure
// for more complex C and CGO builds. You should think at fields inside
// this structure as the enviroment variables you would use and export
// in a bash script (hence the all uppercase naming).
type cBuildEnv struct {
	// BINPATH is the path containing the C and C++ compilers.
	BINPATH string

	// CC is the full path to the C compiler.
	CC string

	// CFLAGS contains the extra CFLAGS to set.
	CFLAGS []string

	// CONFIGURE_HOST is the value to pass to ./configure's --host option.
	CONFIGURE_HOST string

	// DESTDIR is the directory where to install.
	DESTDIR string

	// CXX is the full path to the CXX compiler.
	CXX string

	// CXXFLAGS contains the extra CXXFLAGS to set.
	CXXFLAGS []string

	// GOARCH is the GOARCH we're building for.
	GOARCH string

	// GOARM is the GOARM subarchitecture.
	GOARM string

	// lfdlags contains the LDFLAGS to use when compiling.
	LDFLAGS []string

	// OPENSSL_API_DEFINE is an extra define we need to add on Android.
	OPENSSL_API_DEFINE string

	// OPENSSL_COMPILER is the compiler name for OpenSSL.
	OPENSSL_COMPILER string
}

// cBuildExportEnviron merges the gloval and the local c build environment
// to produce the environment variables to export for the build. More specifically,
// this appends the local variables to the remote variables for any string slice
// type inside [cBuildEnv]. We ignore all the other variables.
func cBuildExportEnviron(global, local *cBuildEnv) *shellx.Envp {
	envp := &shellx.Envp{}

	cflags := append([]string{}, global.CFLAGS...)
	cflags = append(cflags, local.CFLAGS...)
	if len(cflags) > 0 {
		envp.Append("CFLAGS", strings.Join(cflags, " "))
	}

	cxxflags := append([]string{}, global.CXXFLAGS...)
	cxxflags = append(cxxflags, local.CXXFLAGS...)
	if len(cxxflags) > 0 {
		envp.Append("CXXFLAGS", strings.Join(cxxflags, " "))
	}

	ldflags := append([]string{}, global.LDFLAGS...)
	ldflags = append(ldflags, local.LDFLAGS...)
	if len(ldflags) > 0 {
		envp.Append("LDFLAGS", strings.Join(ldflags, " "))
	}

	return envp
}
