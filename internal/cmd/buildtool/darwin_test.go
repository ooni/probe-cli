package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtooltest"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestDarwinBuildAll(t *testing.T) {

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// hasPsiphon indicates whether we should build with psiphon config
		hasPsiphon bool

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name:       "with psiphon config",
		hasPsiphon: true,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{
				"CGO_ENABLED=1",
				"GOARCH=amd64",
				"GOOS=darwin",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-darwin-amd64",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"GOARCH=amd64",
				"GOOS=darwin",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-darwin-amd64",
				"./cmd/ooniprobe",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"GOARCH=arm64",
				"GOOS=darwin",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-darwin-arm64",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"GOARCH=arm64",
				"GOOS=darwin",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-darwin-arm64",
				"./cmd/ooniprobe",
			},
		}},
	}, {
		name:       "without psiphon config",
		hasPsiphon: false,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{
				"CGO_ENABLED=1",
				"GOARCH=amd64",
				"GOOS=darwin",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/miniooni-darwin-amd64",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"GOARCH=amd64",
				"GOOS=darwin",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/ooniprobe-darwin-amd64",
				"./cmd/ooniprobe",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"GOARCH=arm64",
				"GOOS=darwin",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/miniooni-darwin-arm64",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"GOARCH=arm64",
				"GOOS=darwin",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/ooniprobe-darwin-arm64",
				"./cmd/ooniprobe",
			},
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			cc := &buildtooltest.SimpleCommandCollector{}

			deps := &buildtooltest.DependenciesCallCounter{
				HasPsiphon: testcase.hasPsiphon,
			}

			shellxtesting.WithCustomLibrary(cc, func() {
				darwinBuildAll(deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagGolangCheck:           1,
				buildtooltest.TagMaybeCopyPsiphonFiles: 1,
				buildtooltest.TagPsiphonFilesExist:     4,
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
