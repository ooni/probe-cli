package main

import (
	"os"
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

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// goarch is the GOARCH value
		goarch string

		// goarm is the GOARM value
		goarm int64

		// hasPsiphon indicates whether we should build with psiphon config
		hasPsiphon bool

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name:       "build for arm64 where we have the psiphon config",
		goarch:     "arm64",
		goarm:      0,
		hasPsiphon: true,
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"git", "config", "--global", "--add", "safe.directory", "/ooni"},
		}, {
			Env: []string{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/arm64/buildcache",
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/arm64/modcache",
				"CGO_ENABLED=1",
				"GOARCH=arm64",
				"GOOS=linux",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/miniooni-linux-arm64",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/arm64/buildcache",
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/arm64/modcache",
				"CGO_ENABLED=1",
				"GOARCH=arm64",
				"GOOS=linux",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/ooniprobe-linux-arm64",
				"./cmd/ooniprobe",
			},
		}},
	}, {
		name:       "build for amd64 where we don't have the psiphon config",
		goarch:     "amd64",
		goarm:      0,
		hasPsiphon: false,
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"git", "config", "--global", "--add", "safe.directory", "/ooni"},
		}, {
			Env: []string{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/amd64/buildcache",
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/amd64/modcache",
				"CGO_ENABLED=1",
				"GOARCH=amd64",
				"GOOS=linux",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w -extldflags -static",
				"-o", "CLI/miniooni-linux-amd64", "./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/amd64/buildcache",
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/amd64/modcache",
				"CGO_ENABLED=1",
				"GOARCH=amd64",
				"GOOS=linux",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w -extldflags -static",
				"-o", "CLI/ooniprobe-linux-amd64", "./cmd/ooniprobe",
			},
		}},
	}, {
		name:       "build for armv7 where we have the psiphon config",
		goarch:     "arm",
		goarm:      7,
		hasPsiphon: true,
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"git", "config", "--global", "--add", "safe.directory", "/ooni"},
		}, {
			Env: []string{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv7/buildcache",
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv7/modcache",
				"CGO_ENABLED=1",
				"GOARCH=arm",
				"GOARM=7",
				"GOOS=linux",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/miniooni-linux-armv7",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv7/buildcache",
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv7/modcache",
				"CGO_ENABLED=1",
				"GOARCH=arm",
				"GOARM=7",
				"GOOS=linux",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w -extldflags -static", "-o", "CLI/ooniprobe-linux-armv7",
				"./cmd/ooniprobe",
			},
		}},
	}, {
		name:       "build for armv6 where we don't have the psiphon config",
		goarch:     "arm",
		goarm:      6,
		hasPsiphon: false,
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"git", "config", "--global", "--add", "safe.directory", "/ooni"},
		}, {
			Env: []string{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv6/buildcache",
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv6/modcache",
				"CGO_ENABLED=1",
				"GOARCH=arm",
				"GOARM=6",
				"GOOS=linux",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w -extldflags -static",
				"-o", "CLI/miniooni-linux-armv6", "./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"GOCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv6/buildcache",
				"GOMODCACHE=" + cwd + "/GOCACHE/oonibuild/v1/armv6/modcache",
				"CGO_ENABLED=1",
				"GOARCH=arm",
				"GOARM=6",
				"GOOS=linux",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w -extldflags -static",
				"-o", "CLI/ooniprobe-linux-armv6", "./cmd/ooniprobe",
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
				linuxStaticBuilAll(deps, testcase.goarch, testcase.goarm)
			})

			expectCalls := map[string]int{
				buildtooltest.TagGolangCheck:                 1,
				buildtooltest.TagPsiphonMaybeCopyConfigFiles: 1,
				buildtooltest.TagPsiphonFilesExist:           2,
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
