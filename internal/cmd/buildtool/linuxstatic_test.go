package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtooltest"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestLinuxStaticBuildAll(t *testing.T) {

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// The static build also builds the userauth staticlib from source, so we assert
	// its commands too.
	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()
	userauthLib := func(archDir string) string {
		return filepath.Join(faketopdir, "internal", "userauth", "lib", "linux", archDir)
	}
	userauthInc := filepath.Join(faketopdir, "internal", "userauth", "lib", "include")
	tarball := "v0.1.5.tar.gz"
	srcURL := "https://github.com/ooni/ooniprobe-rs/archive/" + tarball

	// userauthExpect returns the userauth build commands for one arch dir.
	userauthExpect := func(archDir string, is32bit bool) []buildtooltest.ExecExpectations {
		cargoEnv := []string{}
		if is32bit {
			cargoEnv = []string{"RUST_LIBC_UNSTABLE_MUSL_V1_2_3=1"}
		}
		return []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"curl", "-fsSLO", srcURL},
		}, {
			Env:  []string{},
			Argv: []string{"tar", "-xf", tarball},
		}, {
			Env:  cargoEnv,
			Argv: []string{"cargo", "build", "-p", "ooniprobe-ffi", "--release"},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", userauthLib(archDir)},
		}, {
			Env: []string{},
			Argv: []string{
				"cp",
				filepath.Join("target", "release", "libuniffi_ooniprobe.a"),
				userauthLib(archDir),
			},
		}, {
			Env:  []string{},
			Argv: []string{"mkdir", "-p", userauthInc},
		}, {
			Env: []string{},
			Argv: []string{
				"cargo", "run", "-p", "cbindgen-gen", "--",
				"--config", filepath.Join("ooniprobe-ffi", "cbindgen.toml"),
				"--lang", "c",
				"--output", filepath.Join(userauthInc, "ooniprobe_userauth.h"),
				filepath.Join("ooniprobe-ffi", "src", "capi.rs"),
			},
		}}
	}

	// gobuild returns the go build command for one product.
	gobuild := func(ooniArch, goarch, goarm, out, pkg string, psiphon bool) buildtooltest.ExecExpectations {
		env := []string{
			"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/" + ooniArch + "/buildcache",
			"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/" + ooniArch + "/modcache",
			"CGO_ENABLED=1",
			"GOARCH=" + goarch,
			"GOOS=linux",
			"CGO_CFLAGS=-D_LARGEFILE64_SOURCE",
		}
		if goarm != "" {
			env = append(env, "GOARM="+goarm) // order does not matter, env is a set
		}
		argv := []string{"go", "build"}
		if psiphon {
			argv = append(argv, "-tags", "ooni_psiphon_config")
		}
		argv = append(argv, "-ldflags", "-s -w -extldflags -static", "-o", out, pkg)
		return buildtooltest.ExecExpectations{Env: env, Argv: argv}
	}

	gitConfig := buildtooltest.ExecExpectations{
		Env:  []string{},
		Argv: []string{"git", "config", "--global", "--add", "safe.directory", "/ooni"},
	}

	var testcases = []struct {
		name       string
		goarch     string
		goarm      int64
		archDir    string
		is32bit    bool
		hasPsiphon bool
		ooniArch   string
		goarmStr   string
	}{{
		name:       "arm64 with the psiphon config (64-bit, no time64)",
		goarch:     "arm64",
		goarm:      0,
		archDir:    "aarch64",
		is32bit:    false,
		hasPsiphon: true,
		ooniArch:   "arm64",
		goarmStr:   "",
	}, {
		name:       "armv7 without the psiphon config (32-bit, time64)",
		goarch:     "arm",
		goarm:      7,
		archDir:    "arm",
		is32bit:    true,
		hasPsiphon: false,
		ooniArch:   "armv7",
		goarmStr:   "7",
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			cc := &buildtooltest.SimpleCommandCollector{}
			deps := &buildtooltest.DependenciesCallCounter{
				HasPsiphon: testcase.hasPsiphon,
			}

			shellxtesting.WithCustomLibrary(cc, func() {
				linuxStaticBuilAll(deps, testcase.goarch, testcase.goarm)
			})

			expectCalls := map[string]int{
				buildtooltest.TagGolangCheck:                 1,
				buildtooltest.TagPsiphonMaybeCopyConfigFiles: 1,
				buildtooltest.TagPsiphonFilesExist:           2,
				buildtooltest.TagAbsoluteCurDir:              1,
				buildtooltest.TagVerifySHA256:                1,
				buildtooltest.TagMustChdir:                   1,
			}
			if diff := cmp.Diff(expectCalls, deps.Counter); diff != "" {
				t.Fatal(diff)
			}

			// expected order: git workaround, then userauth, then the go builds.
			expect := []buildtooltest.ExecExpectations{gitConfig}
			expect = append(expect, userauthExpect(testcase.archDir, testcase.is32bit)...)
			expect = append(expect,
				gobuild(testcase.ooniArch, testcase.goarch, testcase.goarmStr,
					"CLI/miniooni-linux-"+testcase.ooniArch, "./internal/cmd/miniooni", testcase.hasPsiphon),
				gobuild(testcase.ooniArch, testcase.goarch, testcase.goarmStr,
					"CLI/ooniprobe-linux-"+testcase.ooniArch, "./cmd/ooniprobe", testcase.hasPsiphon),
			)

			if err := buildtooltest.CheckManyCommands(cc.Commands, expect); err != nil {
				t.Fatal(err)
			}
		})
	}
}
