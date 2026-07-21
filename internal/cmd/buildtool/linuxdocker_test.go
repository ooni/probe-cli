package main

import (
	"os"
	"os/user"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtooltest"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestLinuxDockerBuildAll(t *testing.T) {
	taggedImageSuffix := time.Now().Format("20060102")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	user, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// ooniArch is the OONI arch value
		ooniArch string

		// goarm is the GOARM value
		goarm int64

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name:     "works as intended for arm64",
		ooniArch: "arm64",
		goarm:    0,
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"docker", "pull", "--platform", "linux/arm64", "golang:1.14.17-alpine"},
		}, {
			Env: []string{},
			Argv: []string{
				"docker", "build", "--platform", "linux/arm64", "-t",
				"oobuild-arm64-" + taggedImageSuffix, "CLI",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"docker", "run", "--platform", "linux/arm64",
				"--user", user.Uid, "-v", cwd + ":/ooni", "-w", "/ooni",
				"oobuild-arm64-" + taggedImageSuffix, "go", "run", "./internal/cmd/buildtool",
				"linux", "static", "--goarm", "0",
			},
		}},
	}, {
		name:     "works as intended for amd64",
		ooniArch: "amd64",
		goarm:    0,
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"docker", "pull", "--platform", "linux/amd64", "golang:1.14.17-alpine"},
		}, {
			Env: []string{},
			Argv: []string{
				"docker", "build", "--platform", "linux/amd64", "-t",
				"oobuild-amd64-" + taggedImageSuffix, "CLI",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"docker", "run", "--platform", "linux/amd64",
				"--user", user.Uid, "-v", cwd + ":/ooni", "-w", "/ooni",
				"oobuild-amd64-" + taggedImageSuffix, "go", "run", "./internal/cmd/buildtool",
				"linux", "static", "--goarm", "0",
			},
		}},
	}, {
		name:     "works as intended for 386",
		ooniArch: "386",
		goarm:    0,
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"docker", "pull", "--platform", "linux/386", "golang:1.14.17-alpine"},
		}, {
			Env: []string{},
			Argv: []string{
				"docker", "build", "--platform", "linux/386", "-t",
				"oobuild-386-" + taggedImageSuffix, "CLI",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"docker", "run", "--platform", "linux/386",
				"--user", user.Uid, "-v", cwd + ":/ooni", "-w", "/ooni",
				"oobuild-386-" + taggedImageSuffix, "go", "run", "./internal/cmd/buildtool",
				"linux", "static", "--goarm", "0",
			},
		}},
	}, {
		name:     "works as intended for armv7",
		ooniArch: "armv7",
		goarm:    0,
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"docker", "pull", "--platform", "linux/arm/v7", "golang:1.14.17-alpine"},
		}, {
			Env: []string{},
			Argv: []string{
				"docker", "build", "--platform", "linux/arm/v7", "-t",
				"oobuild-armv7-" + taggedImageSuffix, "CLI",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"docker", "run", "--platform", "linux/arm/v7",
				"--user", user.Uid, "-v", cwd + ":/ooni", "-w", "/ooni",
				"oobuild-armv7-" + taggedImageSuffix, "go", "run", "./internal/cmd/buildtool",
				"linux", "static", "--goarm", "7",
			},
		}},
	}, {
		name:     "works as intended for armv6",
		ooniArch: "armv6",
		goarm:    0,
		expect: []buildtooltest.ExecExpectations{{
			Env:  []string{},
			Argv: []string{"docker", "pull", "--platform", "linux/arm/v6", "golang:1.14.17-alpine"},
		}, {
			Env: []string{},
			Argv: []string{
				"docker", "build", "--platform", "linux/arm/v6", "-t",
				"oobuild-armv6-" + taggedImageSuffix, "CLI",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"docker", "run", "--platform", "linux/arm/v6",
				"--user", user.Uid, "-v", cwd + ":/ooni", "-w", "/ooni",
				"oobuild-armv6-" + taggedImageSuffix, "go", "run", "./internal/cmd/buildtool",
				"linux", "static", "--goarm", "6",
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
				linuxDockerBuildAll(deps, testcase.ooniArch)
			})

			expectCalls := map[string]int{
				buildtooltest.TagLinuxReadGOVERSION:          1,
				buildtooltest.TagPsiphonMaybeCopyConfigFiles: 1,
				buildtooltest.TagLinuxWriteDockerfile:        1,
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
