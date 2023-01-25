package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestLinuxStaticBuildAll(t *testing.T) {

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	type expectations struct {
		env  map[string]int
		argv []string
	}

	type testspec struct {
		name       string
		goarch     string
		goarm      int64
		hasPsiphon bool
		expect     []expectations
	}

	var testcases = []testspec{{
		name:       "build for arm64 where we have the psiphon config",
		goarch:     "arm64",
		goarm:      0,
		hasPsiphon: true,
		expect: []expectations{{
			env: map[string]int{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/arm64/buildcache":  1,
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/arm64/modcache": 1,
				"CGO_ENABLED=1": 1,
				"GOARCH=arm64":  1,
				"GOOS=linux":    1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/miniooni-linux-arm64",
				"./internal/cmd/miniooni",
			},
		}, {
			env: map[string]int{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/arm64/buildcache":  1,
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/arm64/modcache": 1,
				"CGO_ENABLED=1": 1,
				"GOARCH=arm64":  1,
				"GOOS=linux":    1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/ooniprobe-linux-arm64",
				"./cmd/ooniprobe",
			},
		}},
	}, {
		name:       "build for amd64 where we don't have the psiphon config",
		goarch:     "amd64",
		goarm:      0,
		hasPsiphon: false,
		expect: []expectations{{
			env: map[string]int{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/amd64/buildcache":  1,
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/amd64/modcache": 1,
				"CGO_ENABLED=1": 1,
				"GOARCH=amd64":  1,
				"GOOS=linux":    1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/miniooni-linux-amd64",
				"./internal/cmd/miniooni",
			},
		}, {
			env: map[string]int{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/amd64/buildcache":  1,
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/amd64/modcache": 1,
				"CGO_ENABLED=1": 1,
				"GOARCH=amd64":  1,
				"GOOS=linux":    1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/ooniprobe-linux-amd64",
				"./cmd/ooniprobe",
			},
		}},
	}, {
		name:       "build for armv7 where we have the psiphon config",
		goarch:     "arm",
		goarm:      7,
		hasPsiphon: true,
		expect: []expectations{{
			env: map[string]int{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv7/buildcache":  1,
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv7/modcache": 1,
				"CGO_ENABLED=1": 1,
				"GOARCH=arm":    1,
				"GOARM=7":       1,
				"GOOS=linux":    1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/miniooni-linux-armv7",
				"./internal/cmd/miniooni",
			},
		}, {
			env: map[string]int{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv7/buildcache":  1,
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv7/modcache": 1,
				"CGO_ENABLED=1": 1,
				"GOARCH=arm":    1,
				"GOARM=7":       1,
				"GOOS=linux":    1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/ooniprobe-linux-armv7",
				"./cmd/ooniprobe",
			},
		}},
	}, {
		name:       "build for armv6 where we don't have the psiphon config",
		goarch:     "arm",
		goarm:      6,
		hasPsiphon: false,
		expect: []expectations{{
			env: map[string]int{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv6/buildcache":  1,
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv6/modcache": 1,
				"CGO_ENABLED=1": 1,
				"GOARCH=arm":    1,
				"GOARM=6":       1,
				"GOOS=linux":    1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/miniooni-linux-armv6",
				"./internal/cmd/miniooni",
			},
		}, {
			env: map[string]int{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv6/buildcache":  1,
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv6/modcache": 1,
				"CGO_ENABLED=1": 1,
				"GOARCH=arm":    1,
				"GOARM=6":       1,
				"GOOS=linux":    1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/ooniprobe-linux-armv6",
				"./cmd/ooniprobe",
			},
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			commands := []*exec.Cmd{}
			library := &shellxtesting.Library{
				MockCmdRun: func(c *exec.Cmd) error {
					commands = append(commands, c)
					return nil
				},
				MockLookPath: func(file string) (string, error) {
					return file, nil
				},
			}

			var calledPsiphonMaybeCopyConfigFiles int64
			var calledGolangCheck int64
			deps := &testBuildDeps{
				MockGolangCheck: func() {
					calledGolangCheck++
				},
				MockPsiphonMaybeCopyConfigFiles: func() {
					calledPsiphonMaybeCopyConfigFiles++
				},
				MockPsiphonFilesExist: func() bool {
					return testcase.hasPsiphon
				},
			}

			shellxtesting.WithCustomLibrary(library, func() {
				linuxStaticBuilAll(deps, testcase.goarch, testcase.goarm)
			})

			if calledGolangCheck <= 0 {
				t.Fatal("did not call golangCheck")
			}
			if calledPsiphonMaybeCopyConfigFiles <= 0 {
				t.Fatal("did not call psiphonMaybeConfigFiles")
			}

			if len(commands) != len(testcase.expect)+1 {
				t.Fatal("unexpected number of commands", len(commands))
			}

			command0 := commands[0]
			command0Envs := shellxtesting.RemoveCommonEnvironmentVariables(command0)
			if diff := cmp.Diff(command0Envs, []string{}); diff != "" {
				t.Fatal(diff)
			}
			expectedCommand0Args := []string{
				"git", "config", "--global",
				"--add", "safe.directory", "/ooni",
			}
			if diff := cmp.Diff(command0.Args, expectedCommand0Args); diff != "" {
				t.Fatal(diff)
			}

			for idx := 0; idx < len(testcase.expect); idx++ {
				command := commands[idx+1]
				envs := shellxtesting.RemoveCommonEnvironmentVariables(command)
				gotEnv := map[string]int{}
				for _, env := range envs {
					gotEnv[env]++
				}
				if diff := cmp.Diff(testcase.expect[idx].env, gotEnv); diff != "" {
					t.Fatal(diff)
				}
				gotArgv := shellxtesting.MustArgv(command)
				if diff := cmp.Diff(testcase.expect[idx].argv, gotArgv); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}
