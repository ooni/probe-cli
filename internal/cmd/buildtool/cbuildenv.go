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
	// ANDROID_HOME is the android home variable.
	ANDROID_HOME string

	// ANDROID_NDK_HOME is the android NDK home variable.
	ANDROID_NDK_HOME string

	// AS is the full path to the assembler.
	AS string

	// AR is the full path to the ar tool.
	AR string

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

	// LD is the full path to the linker.
	LD string

	// LDFLAGS contains the LDFLAGS to use when compiling.
	LDFLAGS []string

	// OPENSSL_API_DEFINE is an extra define we need to add on Android.
	OPENSSL_API_DEFINE string

	// OPENSSL_COMPILER is the compiler name for OpenSSL.
	OPENSSL_COMPILER string

	// RANLIB is the path to the ranlib tool.
	RANLIB string

	// STRIP is the path to the strip tool.
	STRIP string
}

// cBuildMerge merges the global and the local [cBuildEnv] to produce a
// new [cBuildEnv] where the following holds:
//
// 1. all the scalar variables come from the global one;
//
// 2. all the slice variables are the ones in the global one with
// appended the ones in the local one.
//
// This kind of merging allows a build rule to include more
// environment variables to CFLAGS, CXXFLAGS, etc.
func cBuildMerge(global, local *cBuildEnv) *cBuildEnv {
	out := &cBuildEnv{
		ANDROID_HOME:       global.ANDROID_HOME,
		ANDROID_NDK_HOME:   global.ANDROID_NDK_HOME,
		AR:                 global.AR,
		AS:                 global.AS,
		BINPATH:            global.BINPATH,
		CC:                 global.CC,
		CFLAGS:             append([]string{}, global.CFLAGS...),
		CONFIGURE_HOST:     global.CONFIGURE_HOST,
		DESTDIR:            global.DESTDIR,
		CXX:                global.CXX,
		CXXFLAGS:           append([]string{}, global.CXXFLAGS...),
		GOARCH:             global.GOARCH,
		GOARM:              global.GOARM,
		LD:                 global.LD,
		LDFLAGS:            append([]string{}, global.LDFLAGS...),
		OPENSSL_API_DEFINE: global.OPENSSL_API_DEFINE,
		OPENSSL_COMPILER:   global.OPENSSL_COMPILER,
		RANLIB:             global.RANLIB,
		STRIP:              global.STRIP,
	}
	out.CFLAGS = append(out.CFLAGS, local.CFLAGS...)
	out.CXXFLAGS = append(out.CXXFLAGS, local.CXXFLAGS...)
	out.LDFLAGS = append(out.LDFLAGS, local.LDFLAGS...)
	return out
}

// cBuildMaybeAppendScalar is a convenience function for appending a
// scalar environment variable to out.
func cBuildMaybeAppend(out *shellx.Envp, name, value string) {
	if value != "" {
		out.Append(name, value)
	}
}

// cBuildExportAutotools exports all the environment variables
// inside the given [cBuildEnv] required by autotools builds.
func cBuildExportAutotools(env *cBuildEnv) *shellx.Envp {
	out := &shellx.Envp{}
	cBuildMaybeAppend(out, "AR", env.AR)
	cBuildMaybeAppend(out, "AS", env.AS)
	cBuildMaybeAppend(out, "CC", env.CC)
	cBuildMaybeAppend(out, "CFLAGS", strings.Join(env.CFLAGS, " "))
	cBuildMaybeAppend(out, "CXX", env.CXX)
	cBuildMaybeAppend(out, "CXXFLAGS", strings.Join(env.CXXFLAGS, " "))
	cBuildMaybeAppend(out, "LD", env.LD)
	cBuildMaybeAppend(out, "LDFLAGS", strings.Join(env.LDFLAGS, " "))
	cBuildMaybeAppend(out, "RANLIB", env.RANLIB)
	cBuildMaybeAppend(out, "STRIP", env.STRIP)
	return out
}

// cBuildExportOpenSSL exports all the environment variables
// inside the given [cBuildEnv] required by OpenSSL builds.
func cBuildExportOpenSSL(env *cBuildEnv) *shellx.Envp {
	out := &shellx.Envp{}
	cBuildMaybeAppend(out, "ANDROID_HOME", env.ANDROID_HOME)
	cBuildMaybeAppend(out, "ANDROID_NDK_HOME", env.ANDROID_NDK_HOME)
	cBuildMaybeAppend(out, "CFLAGS", strings.Join(env.CFLAGS, " "))
	cBuildMaybeAppend(out, "CXXFLAGS", strings.Join(env.CXXFLAGS, " "))
	cBuildMaybeAppend(out, "LDFLAGS", strings.Join(env.LDFLAGS, " "))
	return out
}
