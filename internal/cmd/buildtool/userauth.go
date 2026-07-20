package main

//
// Building the userauth staticlib (ooniprobe-rs) from source.
//

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

const (
	// userauthCrate is the cargo package producing the staticlib.
	userauthCrate = "ooniprobe-ffi"

	// userauthLibName is the name of the staticlib we install.
	userauthLibName = "libuniffi_ooniprobe.a"

	// userauthHeaderName is the name of the FFI header we install.
	userauthHeaderName = "ooniprobe_userauth.h"
)

// userauthOSDir maps a GOOS to the directory name used by ./internal/userauth.
func userauthOSDir(goos string) string {
	switch goos {
	case "darwin":
		return "macos"
	case "linux", "windows":
		return goos
	default:
		panic(fmt.Errorf("userauth: unsupported GOOS: %s", goos))
	}
}

// userauthArchDir maps a GOARCH to the directory name used by ./internal/userauth.
func userauthArchDir(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	case "386":
		return "x86"
	case "arm":
		return "arm"
	default:
		panic(fmt.Errorf("userauth: unsupported GOARCH: %s", goarch))
	}
}

// userauthRustTarget maps a GOOS/GOARCH pair to the Rust target triple we must
// cross compile for. It returns an empty string when we build natively which is the
// case for linux.
func userauthRustTarget(goos, goarch string) string {
	switch goos {
	case "linux":
		return ""
	case "windows":
		switch goarch {
		case "amd64":
			return "x86_64-pc-windows-gnu"
		case "386":
			return "i686-pc-windows-gnu"
		}
	case "darwin":
		switch goarch {
		case "amd64":
			return "x86_64-apple-darwin"
		case "arm64":
			return "aarch64-apple-darwin"
		}
	}
	panic(fmt.Errorf("userauth: unsupported target: %s/%s", goos, goarch))
}

// userauthBuildStaticlib fetches the pinned ooniprobe-rs sources and builds the
// staticlib for the given GOOS/GOARCH, installing it where ./internal/userauth
// expects to find it.
func userauthBuildStaticlib(deps buildtoolmodel.Dependencies, goos, goarch string) {
	log.Infof("building the userauth staticlib for %s/%s", goos, goarch)

	topdir := deps.AbsoluteCurDir() // must be mockable
	work := cdepsMustMkdirTemp()
	restore := cdepsMustChdir(work)
	defer restore()

	cdepsMustFetch("https://github.com/ooni/ooniprobe-rs/archive/v0.1.4.tar.gz")
	deps.VerifySHA256(
		"b80f7a85520c83a7412a954356c855b57eb1c72ee9413da43b90b9f8fb5a2a00",
		"v0.1.4.tar.gz",
	) // must be mockable
	must.Run(log.Log, "tar", "-xf", "v0.1.4.tar.gz")
	_ = deps.MustChdir("ooniprobe-rs-0.1.4") // must be mockable

	// An empty target means we build for the hosw
	rustTarget := userauthRustTarget(goos, goarch)
	libdir := filepath.Join("target", "release")
	if rustTarget != "" {
		must.Run(log.Log, "rustup", "target", "add", rustTarget)
		libdir = filepath.Join("target", rustTarget, "release")
	}

	argv := []string{"build", "-p", userauthCrate, "--release"}
	if rustTarget != "" {
		argv = append(argv, "--target", rustTarget)
	}
	must.Run(log.Log, "cargo", argv...)

	// Install the staticlib where ./internal/userauth's cgo LDFLAGS look for it.
	destdir := filepath.Join(topdir, "internal", "userauth", "lib",
		userauthOSDir(goos), userauthArchDir(goarch))
	must.Run(log.Log, "mkdir", "-p", destdir)
	must.Run(log.Log, "cp", filepath.Join(libdir, userauthLibName), destdir)

	// Generate the FFI header, which is shared by all architectures.
	incdir := filepath.Join(topdir, "internal", "userauth", "lib", "include")
	must.Run(log.Log, "mkdir", "-p", incdir)
	must.Run(log.Log, "cargo", "run", "-p", "cbindgen-gen", "--",
		"--config", filepath.Join(userauthCrate, "cbindgen.toml"),
		"--lang", "c",
		"--output", filepath.Join(incdir, userauthHeaderName),
		filepath.Join(userauthCrate, "src", "capi.rs"),
	)
}

// userauthBuildAll builds the staticlib for every given architecture.
func userauthBuildAll(deps buildtoolmodel.Dependencies, goos string, archs []string) {
	runtimex.Assert(len(archs) > 0, "expected at least one architecture")
	for _, goarch := range archs {
		userauthBuildStaticlib(deps, goos, goarch)
	}
}

// linuxUserauthSubcommand returns the linux userauth subcommand. We build for the
// current architecture only.
func linuxUserauthSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "userauth",
		Short: "Builds the userauth staticlib from source for the current linux architecture",
		Run: func(cmd *cobra.Command, args []string) {
			runtimex.Assert(runtime.GOOS == "linux", "this command requires linux")
			userauthBuildStaticlib(&buildDeps{}, "linux", runtime.GOARCH)
		},
		Args: cobra.NoArgs,
	}
}

// windowsUserauthSubcommand returns the windows userauth subcommand.
func windowsUserauthSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "userauth",
		Short: "Builds the userauth staticlib from source for windows",
		Run: func(cmd *cobra.Command, args []string) {
			userauthBuildAll(&buildDeps{}, "windows", []string{"386", "amd64"})
		},
		Args: cobra.NoArgs,
	}
}

// darwinUserauthSubcommand returns the darwin userauth subcommand.
func darwinUserauthSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "userauth",
		Short: "Builds the userauth staticlib from source for darwin",
		Run: func(cmd *cobra.Command, args []string) {
			userauthBuildAll(&buildDeps{}, "darwin", []string{"amd64", "arm64"})
		},
		Args: cobra.NoArgs,
	}
}
