package main

//
// Allows building C dependencies using Linux
//

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

// linuxCdepsSubcommand returns the linuxCdeps sucommand.
func linuxCdepsSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cdeps {zlib|openssl|libevent|tor} [zlib|openssl|libevent|tor...]",
		Short: "Builds C dependencies on Linux systems (experimental)",
		Run: func(cmd *cobra.Command, args []string) {
			for _, arg := range args {
				linuxCdepsBuildMain(arg, &cdepsDependenciesStdlib{})
			}
		},
		Args: cobra.MinimumNArgs(1),
	}
}

// linuxCdepsBuildMain is the main of the linuxCdeps build.
func linuxCdepsBuildMain(name string, deps cdepsDependencies) {
	runtimex.Assert(
		runtime.GOOS == "linux" && runtime.GOARCH == "amd64",
		"this command requires linux/amd64",
	)
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
	cdenv := &cdepsEnv{
		binpath:       "",
		cc:            "",
		cflags:        cflags,
		cxx:           "",
		cxxflags:      cflags,
		configureHost: "",
		destdir: runtimex.Try1(filepath.Abs(filepath.Join(
			"internal", "libtor", "linux", runtime.GOARCH,
		))),
		goarch:           "",
		goarm:            "",
		ldflags:          []string{},
		openSSLAPIDefine: "",
		openSSLCompiler:  "linux-x86_64",
	}
	switch name {
	case "libevent":
		cdepsLibeventBuildMain(cdenv, deps)
	case "openssl":
		cdepsOpenSSLBuildMain(cdenv, deps)
	case "tor":
		cdepsTorBuildMain(cdenv, deps)
	case "zlib":
		cdepsZlibBuildMain(cdenv, deps)
	default:
		panic(fmt.Errorf("unknown dependency: %s", name))
	}
}
