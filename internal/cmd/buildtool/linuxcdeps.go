package main

//
// Allows building C dependencies using Linux
//

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

// linuxCdepsSubcommand returns the linuxCdeps sucommand.
func linuxCdepsSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cdeps {zlib|openssl|libevent|tor} [zlib|openssl|libevent|tor...]",
		Short: "Builds C dependencies on Linux systems (experimental)",
		Run: func(cmd *cobra.Command, args []string) {
			// Implementation note: perform the check here such that we can
			// run unit test for the building code from any system
			runtimex.Assert(
				runtime.GOOS == "linux" && runtime.GOARCH == "amd64",
				"this command requires linux/amd64",
			)
			for _, arg := range args {
				linuxCdepsBuildMain(arg, &buildDeps{})
			}
		},
		Args: cobra.MinimumNArgs(1),
	}
}

// linuxCdepsBuildMain is the main of the linuxCdeps build.
func linuxCdepsBuildMain(name string, deps buildtoolmodel.Dependencies) {
	cflags := []string{
		// See https://airbus-seclab.github.io/c-compiler-security/
		"-D_FORTIFY_SOURCE=2",
		"-fstack-protector-strong",
		"-fstack-clash-protection",
		"-fPIC", // makes more sense than -fPIE given that we're building a library
		"-fsanitize=bounds",
		"-fsanitize-undefined-trap-on-error",
		"-O2",
	}
	destdir := runtimex.Try1(filepath.Abs(filepath.Join( // must be absolute
		"internal", "libtor", "linux", runtime.GOARCH,
	)))
	globalEnv := &cBuildEnv{
		ANDROID_HOME:       "",
		ANDROID_NDK_ROOT:   "",
		AR:                 "",
		BINPATH:            "",
		CC:                 "",
		CFLAGS:             cflags,
		CONFIGURE_HOST:     "",
		DESTDIR:            destdir,
		CXX:                "",
		CXXFLAGS:           cflags,
		GOARCH:             "",
		GOARM:              "",
		LD:                 "",
		LDFLAGS:            []string{},
		OPENSSL_API_DEFINE: "",
		OPENSSL_COMPILER:   "linux-x86_64",
		RANLIB:             "",
		STRIP:              "",
	}
	switch name {
	case "libevent":
		cdepsLibeventBuildMain(globalEnv, deps)
	case "openssl":
		cdepsOpenSSLBuildMain(globalEnv, deps)
	case "tor":
		cdepsTorBuildMain(globalEnv, deps)
	case "zlib":
		cdepsZlibBuildMain(globalEnv, deps)
	default:
		panic(fmt.Errorf("unknown dependency: %s", name))
	}
}
