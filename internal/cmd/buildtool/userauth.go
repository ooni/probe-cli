package main

//
// Building the userauth staticlib (ooniprobe-rs) from source.
//

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

const (
	// userauthCrate is the cargo package producing the staticlib.
	userauthCrate = "ooniprobe-ffi"

	// userauthLibName is the name of the staticlib we install.
	userauthLibName = "libuniffi_ooniprobe.a"

	// userauthHeaderName is the name of the FFI header we install.
	userauthHeaderName = "ooniprobe_userauth.h"

	// userauthRustTargetEnv, when set, is the Rust target used to compile the staticlib.
	userauthRustTargetEnv = "USERAUTH_RUST_TARGET"

	// userauthVersion is the ooniprobe-rs release we use for BOTH the from-source
	// build and the prebuilt bundle download.
	userauthVersion = "0.1.5"

	// userauthSourceSHA256 pins the source tarball for userauthVersion.
	userauthSourceSHA256 = "629aff29a75592280ec65c51ad2e22520bae89b26cd403b3a7fafe36d808b751"

	// userauthFromSourceEnv, when set to "1", makes the userauth subcommand build the
	// staticlib from source instead of downloading the prebuilt bundle.
	userauthFromSourceEnv = "USERAUTH_FROM_SOURCE"
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
		return os.Getenv(userauthRustTargetEnv)
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

// userauthEnvp returns the environment for the cargo build.
func userauthEnvp(goos, goarch string) *shellx.Envp {
	envp := &shellx.Envp{}

	if goos == "windows" {
		switch goarch {
		case "386":
			envp.Append("CC", windowsMingw386Compiler)
			envp.Append("CXX", windowsMingw386Cxx)

			// Force a 64-bit long double. Otherwise bindgen parses the mingw headers
			// as having a 12-byte `_LONGDOUBLE` while the type it generates is 8 bytes, and
			// the resulting layout assertion fails to compile.
			const longDouble = "-mlong-double-64"
			envp.Append("CFLAGS", longDouble)
			envp.Append("CXXFLAGS", longDouble)
			envp.Append("BINDGEN_EXTRA_CLANG_ARGS", longDouble)
		case "amd64":
			envp.Append("CC", windowsMingwAmd64Compiler)
			envp.Append("CXX", windowsMingwAmd64Cxx)
		}

		return envp
	}

	// The libc crate still defaults to musl 1.1 semantics, where time_t is 32 bits
	// on 32-bit targets. Alpine ships musl 1.2, which widened time_t to 64 bits and
	// exposed the wider functions under __*_time64 names.
	// https://github.com/rust-lang/libc/issues/1848
	if goos == "linux" && (goarch == "386" || goarch == "arm") {
		envp.Append("RUST_LIBC_UNSTABLE_MUSL_V1_2_3", "1")
	}
	return envp
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

	tarball := "v" + userauthVersion + ".tar.gz"
	cdepsMustFetch("https://github.com/ooni/ooniprobe-rs/archive/" + tarball)
	deps.VerifySHA256(userauthSourceSHA256, tarball) // must be mockable
	must.Run(log.Log, "tar", "-xf", tarball)
	_ = deps.MustChdir("ooniprobe-rs-" + userauthVersion) // must be mockable

	rustTarget := userauthRustTarget(goos, goarch)
	libdir := filepath.Join("target", "release")
	if rustTarget != "" {
		must.Run(log.Log, "rustup", "target", "add", rustTarget)
		libdir = filepath.Join("target", rustTarget, "release")
	}

	envp := userauthEnvp(goos, goarch)
	argv := []string{"build", "-p", userauthCrate, "--release"}
	if rustTarget != "" {
		argv = append(argv, "--target", rustTarget)
	}
	cdepsMustRunWithDefaultConfig(envp, "cargo", argv...)

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

// userauthFromSource reports whether the userauth subcommand should build the
// staticlib from source rather than downloading the prebuilt bundle.
func userauthFromSource() bool {
	return os.Getenv(userauthFromSourceEnv) == "1"
}

// userauthDownloadPrebuilt downloads the prebuilt staticlib bundle for the pinned
// userauthVersion and extracts it where ./internal/userauth expects.
func userauthDownloadPrebuilt(deps buildtoolmodel.Dependencies) {
	log.Infof("downloading the prebuilt userauth staticlib bundle v%s", userauthVersion)

	topdir := deps.AbsoluteCurDir() // must be mockable
	work := cdepsMustMkdirTemp()
	restore := cdepsMustChdir(work)
	defer restore()

	cdepsMustFetch("https://github.com/ooni/ooniprobe-rs/releases/download/v" +
		userauthVersion + "/staticlib.tar.gz")

	// The bundle has lib/ at its root, so extracting into ./internal/userauth lands
	// the files under ./internal/userauth/lib/<os>/<arch>/ where cgo looks for them.
	dest := filepath.Join(topdir, "internal", "userauth")
	must.Run(log.Log, "tar", "-xzf", "staticlib.tar.gz", "-C", dest)
}

// linuxUserauthSubcommand returns the linux userauth subcommand. We build for the
// current architecture only.
func linuxUserauthSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "userauth",
		Short: "Builds the userauth staticlib from source for the current linux architecture",
		Run: func(cmd *cobra.Command, args []string) {
			if !userauthFromSource() {
				userauthDownloadPrebuilt(&buildDeps{})
				return
			}
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
			if !userauthFromSource() {
				userauthDownloadPrebuilt(&buildDeps{})
				return
			}
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
			if !userauthFromSource() {
				userauthDownloadPrebuilt(&buildDeps{})
				return
			}
			userauthBuildAll(&buildDeps{}, "darwin", []string{"amd64", "arm64"})
		},
		Args: cobra.NoArgs,
	}
}
