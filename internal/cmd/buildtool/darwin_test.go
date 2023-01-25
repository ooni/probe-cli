package main

import (
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestDarwinBuildAll(t *testing.T) {

	type expectations struct {
		env  map[string]int
		argv []string
	}

	type testspec struct {
		name       string
		hasPsiphon bool
		expect     []expectations
	}

	var testcases = []testspec{{
		name:       "build where we have the psiphon config",
		hasPsiphon: true,
		expect: []expectations{{
			env: map[string]int{
				"CGO_ENABLED=1": 1,
				"GOARCH=amd64":  1,
				"GOOS=darwin":   1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-darwin-amd64",
				"./internal/cmd/miniooni",
			},
		}, {
			env: map[string]int{
				"CGO_ENABLED=1": 1,
				"GOARCH=amd64":  1,
				"GOOS=darwin":   1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-darwin-amd64",
				"./cmd/ooniprobe",
			},
		}, {
			env: map[string]int{
				"CGO_ENABLED=1": 1,
				"GOARCH=arm64":  1,
				"GOOS=darwin":   1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-darwin-arm64",
				"./internal/cmd/miniooni",
			},
		}, {
			env: map[string]int{
				"CGO_ENABLED=1": 1,
				"GOARCH=arm64":  1,
				"GOOS=darwin":   1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-darwin-arm64",
				"./cmd/ooniprobe",
			},
		}},
	}, {
		name:       "build where we don't have the psiphon config",
		hasPsiphon: false,
		expect: []expectations{{
			env: map[string]int{
				"CGO_ENABLED=1": 1,
				"GOARCH=amd64":  1,
				"GOOS=darwin":   1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-darwin-amd64",
				"./internal/cmd/miniooni",
			},
		}, {
			env: map[string]int{
				"CGO_ENABLED=1": 1,
				"GOARCH=amd64":  1,
				"GOOS=darwin":   1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-darwin-amd64",
				"./cmd/ooniprobe",
			},
		}, {
			env: map[string]int{
				"CGO_ENABLED=1": 1,
				"GOARCH=arm64":  1,
				"GOOS=darwin":   1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-darwin-arm64",
				"./internal/cmd/miniooni",
			},
		}, {
			env: map[string]int{
				"CGO_ENABLED=1": 1,
				"GOARCH=arm64":  1,
				"GOOS=darwin":   1,
			},
			argv: []string{
				runtimex.Try1(exec.LookPath("go")),
				"build",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-darwin-arm64",
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
				MockWindowsMingwCheck: func() {
					panic("should not be called")
				},
			}

			shellxtesting.WithCustomLibrary(library, func() {
				darwinBuildAll(deps)
			})

			if calledGolangCheck <= 0 {
				t.Fatal("did not call golangCheck")
			}
			if calledPsiphonMaybeCopyConfigFiles <= 0 {
				t.Fatal("did not call psiphonMaybeConfigFiles")
			}

			if len(commands) != len(testcase.expect) {
				t.Fatal("unexpected number of commands", len(commands))
			}
			for idx := 0; idx < len(testcase.expect); idx++ {
				command := commands[idx]
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
