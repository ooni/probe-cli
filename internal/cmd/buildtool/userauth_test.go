package main

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtooltest"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestUserauthBuildStaticlib(t *testing.T) {
	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()

	// libdir returns the directory where we install the staticlib
	libdir := func(osdir, archdir string) string {
		return filepath.Join(faketopdir, "internal", "userauth", "lib", osdir, archdir)
	}
	incdir := filepath.Join(faketopdir, "internal", "userauth", "lib", "include")
	header := filepath.Join(incdir, "ooniprobe_userauth.h")
	tarball := "v0.1.5.tar.gz"
	srcURL := "https://github.com/ooni/ooniprobe-rs/archive/" + tarball

	// cbindgen returns the header generation command, which does not vary
	cbindgen := []string{
		"cargo", "run", "-p", "cbindgen-gen", "--",
		"--config", filepath.Join("ooniprobe-ffi", "cbindgen.toml"),
		"--lang", "c",
		"--output", header,
		filepath.Join("ooniprobe-ffi", "src", "capi.rs"),
	}

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// goos and goarch are the target we build for
		goos, goarch string

		// expect contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		// On Linux we build natively, because this command runs inside a
		// container whose architecture and libc already are the target.
		name:   "linux builds natively without a rust target",
		goos:   "linux",
		goarch: "amd64",
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"curl", "-fsSLO", srcURL},
		}, {
			Env:  []string{},
			Argv: []string{"tar", "-xf", tarball},
		}, {
			Env:  []string{},
			Argv: []string{"cargo", "build", "-p", "ooniprobe-ffi", "--release"},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", libdir("linux", "x86_64")},
		}, {
			Env: []string{},
			Argv: []string{
				"cp",
				filepath.Join("target", "release", "libuniffi_ooniprobe.a"),
				libdir("linux", "x86_64"),
			},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", incdir},
		}, {
			Env:  []string{},
			Argv: cbindgen,
		}},
	}, {
		// On 32-bit musl we must tell the libc crate that time_t is 64 bits,
		// because alpine ships musl 1.2 while the crate defaults to musl 1.1.
		name:   "linux/386 asks the libc crate for musl 1.2.3 semantics",
		goos:   "linux",
		goarch: "386",
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"curl", "-fsSLO", srcURL},
		}, {
			Env:  []string{},
			Argv: []string{"tar", "-xf", tarball},
		}, {
			Env:  []string{"RUST_LIBC_UNSTABLE_MUSL_V1_2_3=1"},
			Argv: []string{"cargo", "build", "-p", "ooniprobe-ffi", "--release"},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", libdir("linux", "x86")},
		}, {
			Env: []string{},
			Argv: []string{
				"cp",
				filepath.Join("target", "release", "libuniffi_ooniprobe.a"),
				libdir("linux", "x86"),
			},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", incdir},
		}, {
			Env:  []string{},
			Argv: cbindgen,
		}},
	}, {
		// armv6 and armv7 are both GOARCH=arm and need the same flag.
		name:   "linux/arm asks the libc crate for musl 1.2.3 semantics",
		goos:   "linux",
		goarch: "arm",
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"curl", "-fsSLO", srcURL},
		}, {
			Env:  []string{},
			Argv: []string{"tar", "-xf", tarball},
		}, {
			Env:  []string{"RUST_LIBC_UNSTABLE_MUSL_V1_2_3=1"},
			Argv: []string{"cargo", "build", "-p", "ooniprobe-ffi", "--release"},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", libdir("linux", "arm")},
		}, {
			Env: []string{},
			Argv: []string{
				"cp",
				filepath.Join("target", "release", "libuniffi_ooniprobe.a"),
				libdir("linux", "arm"),
			},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", incdir},
		}, {
			Env:  []string{},
			Argv: cbindgen,
		}},
	}, {
		// Windows and darwin cross compile, so they must add and pass a target.
		name:   "windows/386 cross compiles with a rust target",
		goos:   "windows",
		goarch: "386",
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"curl", "-fsSLO", srcURL},
		}, {
			Env:  []string{},
			Argv: []string{"tar", "-xf", tarball},
		}, {
			Env:  []string{},
			Argv: []string{"rustup", "target", "add", "i686-pc-windows-gnu"},
		}, {
			Env: []string{
				"CC=i686-w64-mingw32-gcc",
				"CXX=i686-w64-mingw32-g++",
				"CFLAGS=-mlong-double-64",
				"CXXFLAGS=-mlong-double-64",
				"BINDGEN_EXTRA_CLANG_ARGS=-mlong-double-64",
			},
			Argv: []string{
				"cargo", "build", "-p", "ooniprobe-ffi", "--release",
				"--target", "i686-pc-windows-gnu",
			},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", libdir("windows", "x86")},
		}, {
			Env: []string{},
			Argv: []string{
				"cp",
				filepath.Join("target", "i686-pc-windows-gnu", "release", "libuniffi_ooniprobe.a"),
				libdir("windows", "x86"),
			},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", incdir},
		}, {
			Env:  []string{},
			Argv: cbindgen,
		}},
	}, {
		// Darwin is spelled "macos" inside ./internal/userauth.
		name:   "darwin/arm64 installs under the macos directory",
		goos:   "darwin",
		goarch: "arm64",
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"curl", "-fsSLO", srcURL},
		}, {
			Env:  []string{},
			Argv: []string{"tar", "-xf", tarball},
		}, {
			Env:  []string{},
			Argv: []string{"rustup", "target", "add", "aarch64-apple-darwin"},
		}, {
			Env: []string{},
			Argv: []string{
				"cargo", "build", "-p", "ooniprobe-ffi", "--release",
				"--target", "aarch64-apple-darwin",
			},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", libdir("macos", "aarch64")},
		}, {
			Env: []string{},
			Argv: []string{
				"cp",
				filepath.Join("target", "aarch64-apple-darwin", "release", "libuniffi_ooniprobe.a"),
				libdir("macos", "aarch64"),
			},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", incdir},
		}, {
			Env:  []string{},
			Argv: cbindgen,
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			cc := &buildtooltest.SimpleCommandCollector{}
			deps := &buildtooltest.DependenciesCallCounter{}

			shellxtesting.WithCustomLibrary(cc, func() {
				userauthBuildStaticlib(deps, testcase.goos, testcase.goarch)
			})

			expectCalls := map[string]int{
				buildtooltest.TagAbsoluteCurDir: 1,
				buildtooltest.TagVerifySHA256:   1,
				buildtooltest.TagMustChdir:      1,
			}
			if diff := cmp.Diff(expectCalls, deps.Counter); diff != "" {
				t.Fatal(diff)
			}

			if err := buildtooltest.CheckManyCommands(cc.Commands, testcase.expect); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUserauthDownloadPrebuilt(t *testing.T) {
	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()

	cc := &buildtooltest.SimpleCommandCollector{}
	deps := &buildtooltest.DependenciesCallCounter{}

	shellxtesting.WithCustomLibrary(cc, func() {
		userauthDownloadPrebuilt(deps)
	})

	// We only resolve the top directory; the download does not verify a SHA256 or
	// chdir through the mockable dependency.
	expectCalls := map[string]int{
		buildtooltest.TagAbsoluteCurDir: 1,
	}
	if diff := cmp.Diff(expectCalls, deps.Counter); diff != "" {
		t.Fatal(diff)
	}

	expect := []buildtooltest.ExecExpectations{{
		Env: []string{},
		Argv: []string{
			"curl", "-fsSLO",
			"https://github.com/ooni/ooniprobe-rs/releases/download/v0.1.5/staticlib.tar.gz",
		},
	}, {
		Env: []string{},
		Argv: []string{
			"tar", "-xzf", "staticlib.tar.gz", "-C",
			filepath.Join(faketopdir, "internal", "userauth"),
		},
	}}
	if err := buildtooltest.CheckManyCommands(cc.Commands, expect); err != nil {
		t.Fatal(err)
	}
}
