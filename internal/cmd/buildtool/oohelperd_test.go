package main

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtooltest"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestOohelperdBuildAndMaybeDeploy(t *testing.T) {

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// deploy indicates whether we also want to deploy.
		deploy bool

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name:   "oohelperd build without automatic deployment",
		deploy: false,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{
				"CGO_ENABLED=0",
				"GOOS=linux",
				"GOARCH=amd64",
			},
			Argv: []string{
				"go", "build",
				"-o", filepath.Join("CLI", "oohelperd-linux-amd64"),
				"-tags", "netgo",
				"-ldflags", "-s -w -extldflags -static",
				"./internal/cmd/oohelperd",
			},
		}},
	}, {
		name:   "oohelperd build with automatic deployment",
		deploy: true,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{
				"CGO_ENABLED=0",
				"GOOS=linux",
				"GOARCH=amd64",
			},
			Argv: []string{
				"go", "build",
				"-o", filepath.Join("CLI", "oohelperd-linux-amd64"),
				"-tags", "netgo",
				"-ldflags", "-s -w -extldflags -static",
				"./internal/cmd/oohelperd",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"scp", filepath.Join("CLI", "oohelperd-linux-amd64"),
				"0.th.ooni.org:oohelperd",
			},
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			cc := &buildtooltest.SimpleCommandCollector{}

			deps := &buildtooltest.DependenciesCallCounter{
				HasPsiphon: false,
			}

			shellxtesting.WithCustomLibrary(cc, func() {
				oohelperdBuildAndMaybeDeploy(deps, testcase.deploy)
			})

			expectCalls := map[string]int{
				buildtooltest.TagGolangCheck: 1,
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
