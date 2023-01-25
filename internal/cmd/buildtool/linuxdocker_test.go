package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestLinuxDockerBuildAll(t *testing.T) {

	type testspec struct {
		name     string
		ooniArch string
	}

	var testcases = []testspec{{
		name:     "works as intended for arm64 without need to rebuild",
		ooniArch: "arm64",
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			commands := []*exec.Cmd{}
			library := &shellxtesting.Library{
				MockCmdRun: func(c *exec.Cmd) error {
					commands = append(commands, c)
					return nil
				},
				MockCmdOutput: func(c *exec.Cmd) ([]byte, error) {
					return nil, nil
				},
				MockLookPath: func(file string) (string, error) {
					return file, nil
				},
			}

			expectedGOVERSION := "1.14" // it does not really matter

			var calledPsiphonMaybeCopyConfigFiles int64
			var calledWriteDockerfile int64
			deps := &testBuildDeps{
				MockLinuxWriteDockerfile: func(filename string, content []byte, mode fs.FileMode) {
					calledWriteDockerfile++
				},
				MockPsiphonMaybeCopyConfigFiles: func() {
					calledPsiphonMaybeCopyConfigFiles++
				},
				MockLinuxReadGOVERSION: func(filename string) []byte {
					return []byte(expectedGOVERSION + "\n")
				},
			}

			shellxtesting.WithCustomLibrary(library, func() {
				linuxDockerBuildAll(deps, testcase.ooniArch)
			})

			if calledPsiphonMaybeCopyConfigFiles <= 0 {
				t.Fatal("did not call psiphonMaybeConfigFiles")
			}
			if calledWriteDockerfile <= 0 {
				t.Fatal("did not call writeDockerfile")
			}

			const expectedCommands = 3
			if len(commands) != expectedCommands {
				t.Fatal("expected the code to run", expectedCommands, "commands, got", len(commands))
			}
			testDockerExpectDockerPull(t, testcase.ooniArch, commands[0])
			testDockerExpectDockerBuild(t, testcase.ooniArch, commands[1])
			testDockerExpectDockerRun(t, testcase.ooniArch, commands[2])
		})
	}
}

// testDockerArch maps the ooniArch to the correct dockerArch
var testDockerArch = map[string]string{
	"arm64": "arm64",
	"armv7": "arm/v7",
	"armv6": "arm/v6",
	"386":   "386",
	"amd64": "amd64",
}

// testDockerGoArm maps the ooniArch to the correct GOARM
var testDockerGoArm = map[string]string{
	"arm64": "0",
	"armv7": "7",
	"armv6": "6",
	"386":   "0",
	"amd64": "0",
}

func testCompareArgv(expected, got []string) error {
	if len(expected) != len(got) {
		return fmt.Errorf("expected %d entries but got %d", len(expected), len(got))
	}
	if len(got) < 1 {
		return errors.New("expected at least one entry")
	}
	if !strings.HasSuffix(got[0], expected[0]) {
		return fmt.Errorf("expected %s prefix but got %s", expected[0], got[0])
	}
	for idx := 1; idx < len(got); idx++ {
		if got[idx] != expected[idx] {
			return fmt.Errorf("entry %d: expected %s, but got %s", idx, expected[idx], got[idx])
		}
	}
	return nil
}

func testDockerExpectDockerPull(t *testing.T, ooniArch string, cmd *exec.Cmd) {
	envs := shellxtesting.RemoveCommonEnvironmentVariables(cmd)
	if diff := cmp.Diff(envs, []string{}); diff != "" {
		t.Fatal(diff)
	}
	expectedArgv := []string{
		"docker", "pull",
		"--platform", "linux/" + testDockerArch[ooniArch],
		"golang:1.14-alpine",
	}
	if err := testCompareArgv(expectedArgv, cmd.Args); err != nil {
		t.Fatal(err)
	}
}

func testDockerExpectDockerBuild(t *testing.T, ooniArch string, cmd *exec.Cmd) {
	envs := shellxtesting.RemoveCommonEnvironmentVariables(cmd)
	if diff := cmp.Diff(envs, []string{}); diff != "" {
		t.Fatal(diff)
	}
	taggedImage := fmt.Sprintf("oobuild-%s-%s", ooniArch, time.Now().Format("20060102"))
	expectedArgv := []string{
		"docker", "build", "--platform", "linux/" + testDockerArch[ooniArch],
		"-t", taggedImage, "CLI",
	}
	if err := testCompareArgv(expectedArgv, cmd.Args); err != nil {
		t.Fatal(err)
	}
}

func testDockerExpectDockerRun(t *testing.T, ooniArch string, cmd *exec.Cmd) {
	envs := shellxtesting.RemoveCommonEnvironmentVariables(cmd)
	if diff := cmp.Diff(envs, []string{}); diff != "" {
		t.Fatal(diff)
	}
	user, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	taggedImage := fmt.Sprintf("oobuild-%s-%s", ooniArch, time.Now().Format("20060102"))
	expectedArgv := []string{
		"docker", "run", "--platform", "linux/" + testDockerArch[ooniArch],
		"--user", user.Uid, "-v", cwd + ":/ooni", "-w", "/ooni",
		taggedImage, "go", "run", "./internal/cmd/buildtool",
		"linux-static", "--goarm", testDockerGoArm[ooniArch],
	}
	if err := testCompareArgv(expectedArgv, cmd.Args); err != nil {
		t.Fatal(err)
	}
}
