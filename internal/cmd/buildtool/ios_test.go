package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtooltest"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestIOSBuildGomobile(t *testing.T) {

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
			Env: []string{},
			Argv: []string{
				"go", "install", "golang.org/x/mobile/cmd/gomobile@latest",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"gomobile", "init",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"go", "get", "-d", "golang.org/x/mobile/cmd/gomobile",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"gomobile", "bind", "-target", "ios",
				"-o", "MOBILE/ios/oonimkall.xcframework",
				"-tags", "ooni_psiphon_config,ooni_libtor",
				"-ldflags", "-s -w",
				"./pkg/oonimkall",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"go", "mod", "tidy",
			},
		}},
	}, {
		name:       "without psiphon config",
		hasPsiphon: false,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"go", "install", "golang.org/x/mobile/cmd/gomobile@latest",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"gomobile", "init",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"go", "get", "-d", "golang.org/x/mobile/cmd/gomobile",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"gomobile", "bind", "-target", "ios",
				"-o", "MOBILE/ios/oonimkall.xcframework",
				"-tags", "ooni_libtor",
				"-ldflags", "-s -w", "./pkg/oonimkall",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"go", "mod", "tidy",
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
				iosBuildGomobile(deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagGOPATH:                      1,
				buildtooltest.TagGolangCheck:                 1,
				buildtooltest.TagPsiphonMaybeCopyConfigFiles: 1,
				buildtooltest.TagPsiphonFilesExist:           1,
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
