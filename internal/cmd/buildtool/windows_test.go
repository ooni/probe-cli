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

func TestWindowsMingwParseVersion(t *testing.T) {
	// Different vendors print different banners, so make sure we cope with
	// every flavour of mingw-w64 we build with.
	for _, tc := range []struct {
		name, firstLine, expect string
	}{{
		name:      "msys2",
		firstLine: "x86_64-w64-mingw32-gcc.exe (Rev5, Built by MSYS2 project) 16.1.0",
		expect:    "16.1.0",
	}, {
		name:      "debian",
		firstLine: "x86_64-w64-mingw32-gcc (GCC) 10-win32 20220324",
		expect:    "10-win32",
	}, {
		name:      "homebrew",
		firstLine: "x86_64-w64-mingw32-gcc (GCC) 15.1.0",
		expect:    "15.1.0",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			if got := windowsMingwParseVersion(tc.firstLine); got != tc.expect {
				t.Fatalf("got %q, expected %q", got, tc.expect)
			}
		})
	}
}
