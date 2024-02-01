package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtooltest"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestGenericBuildPackage(t *testing.T) {

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// product is the product to build
		product *product

		// hasPsiphon indicates whether we should build with psiphon config
		hasPsiphon bool

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name:       "miniooni build with psiphon",
		product:    productMiniooni,
		hasPsiphon: true,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "./internal/cmd/miniooni",
			},
		}},
	}, {
		name:       "miniooni build without psiphon",
		product:    productMiniooni,
		hasPsiphon: false,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "./internal/cmd/miniooni",
			},
		}},
	}, {
		name:       "ooniprobe build with psiphon",
		product:    productOoniprobe,
		hasPsiphon: true,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "./cmd/ooniprobe",
			},
		}},
	}, {
		name:       "ooniprobe build without psiphon",
		product:    productOoniprobe,
		hasPsiphon: false,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "./cmd/ooniprobe",
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
				genericBuildPackage(deps, testcase.product)
			})

			expectCalls := map[string]int{
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

func TestCheckGenericBuildLibrary(t *testing.T) {
	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// product is the product to build
		product *product

		// os is the runtime.GOOS value to use
		os string

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name:    "libooniengine build on windows",
		product: productLibooniengine,
		os:      "windows",
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"go", "build", "-buildmode", "c-shared",
				"-o", "libooniengine.dll", "./internal/libooniengine",
			},
		}},
	}, {
		name:    "libooniengine build on linux",
		product: productLibooniengine,
		os:      "linux",
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"go", "build", "-buildmode", "c-shared",
				"-o", "libooniengine.so", "./internal/libooniengine",
			},
		}},
	}, {
		name:    "libooniengine build on darwin",
		product: productLibooniengine,
		os:      "darwin",
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"go", "build", "-buildmode", "c-shared",
				"-o", "libooniengine.dylib", "./internal/libooniengine",
			},
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			cc := &buildtooltest.SimpleCommandCollector{}

			deps := &buildtooltest.DependenciesCallCounter{
				OS: testcase.os,
			}

			shellxtesting.WithCustomLibrary(cc, func() {
				genericBuildLibrary(deps, testcase.product)
			})

			expectCalls := map[string]int{
				buildtooltest.TagGolangCheck: 1,
				buildtooltest.TagGOOS:        1,
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
