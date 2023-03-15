package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtooltest"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestWindowsBuildAll(t *testing.T) {

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
		name:       "build where we have the psiphon config",
		hasPsiphon: true,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{
				"CC=i686-w64-mingw32-gcc",
				"CGO_ENABLED=1",
				"GOARCH=386",
				"GOOS=windows",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-windows-386.exe",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CC=i686-w64-mingw32-gcc",
				"CGO_ENABLED=1",
				"GOARCH=386",
				"GOOS=windows",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-windows-386.exe",
				"./cmd/ooniprobe",
			},
		}, {
			Env: []string{
				"CC=x86_64-w64-mingw32-gcc",
				"CGO_ENABLED=1",
				"GOARCH=amd64",
				"GOOS=windows",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-windows-amd64.exe",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CC=x86_64-w64-mingw32-gcc",
				"CGO_ENABLED=1",
				"GOARCH=amd64",
				"GOOS=windows",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-windows-amd64.exe",
				"./cmd/ooniprobe",
			},
		}},
	}, {
		name:       "build where we don't have the psiphon config",
		hasPsiphon: false,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{
				"CC=i686-w64-mingw32-gcc",
				"CGO_ENABLED=1",
				"GOARCH=386",
				"GOOS=windows",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/miniooni-windows-386.exe",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CC=i686-w64-mingw32-gcc",
				"CGO_ENABLED=1",
				"GOARCH=386",
				"GOOS=windows",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/ooniprobe-windows-386.exe",
				"./cmd/ooniprobe",
			},
		}, {
			Env: []string{
				"CC=x86_64-w64-mingw32-gcc",
				"CGO_ENABLED=1",
				"GOARCH=amd64",
				"GOOS=windows",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/miniooni-windows-amd64.exe",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CC=x86_64-w64-mingw32-gcc",
				"CGO_ENABLED=1",
				"GOARCH=amd64",
				"GOOS=windows",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/ooniprobe-windows-amd64.exe",
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
				windowsBuildAll(deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagGolangCheck:                 1,
				buildtooltest.TagPsiphonMaybeCopyConfigFiles: 1,
				buildtooltest.TagPsiphonFilesExist:           4,
				buildtooltest.TagWindowsMingwCheck:           1,
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
