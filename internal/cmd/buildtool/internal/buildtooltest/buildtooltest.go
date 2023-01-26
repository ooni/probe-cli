// Package buildtooltest contains code for testing buildtool.
package buildtooltest

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
	"golang.org/x/sys/execabs"
)

// ExecExpectations describes what we would expect to see
// when building in terms of executed subcommands.
type ExecExpectations struct {
	// Env contains the environment variables we would expect to see.
	Env []string

	// Argv contains the Argv we would expect to see. The first
	// argument is matched as a suffix, to account for various
	// utable paths (e.g., /bin and /usr/bin). All the other
	// arguments are matched exactly.
	Argv []string
}

// CompareArgv compares the expected argv with the one we've got
// and returns an explanatory error if they do not match.
func CompareArgv(expected, got []string) error {
	if len(expected) != len(got) {
		return fmt.Errorf("expected %d entries but got %d", len(expected), len(got))
	}
	runtimex.Assert(len(got) >= 1, "too few entries")
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

// CompareEnv compares the expected environment with the one we've got
// and returns an explanatory error if they do not match.
func CompareEnv(expected, got []string) error {
	const (
		weExpected = 1 << iota
		weGot
	)
	uniq := map[string]int{}
	for _, entry := range expected {
		uniq[entry] |= weExpected
	}
	for _, entry := range got {
		uniq[entry] |= weGot
	}
	var issues []string
	for value, flags := range uniq {
		runtimex.Assert(flags&(^(weExpected|weGot)) == 0, "extra flags")
		switch flags {
		case weExpected | weGot:
			// nothing
		case weGot:
			issues = append(issues, fmt.Sprintf("* we got %s, which we didn't expect", value))
		case weExpected:
			issues = append(issues, fmt.Sprintf("* we expected but did not see %s", value))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "\n"))
	}
	return nil
}

// CheckSingleCommand checks whether the given command contains an argv and
// environ variables matching the expectations we had.
func CheckSingleCommand(cmd *execabs.Cmd, tee ExecExpectations) error {
	if err := CompareArgv(tee.Argv, shellxtesting.MustArgv(cmd)); err != nil {
		return err
	}
	if err := CompareEnv(tee.Env, shellxtesting.CmdEnvironMinusOsEnviron(cmd)); err != nil {
		return err
	}
	return nil
}

// CheckManyCommands applies CheckSingleCommand for each command
// by comparing it with the matching expectation.
func CheckManyCommands(cmd []*execabs.Cmd, tee []ExecExpectations) error {
	if len(cmd) != len(tee) {
		return fmt.Errorf("expected to see %d commands, got %d", len(tee), len(cmd))
	}
	runtimex.Assert(len(cmd) > 0, "expected to see at least one command")
	for idx := 0; idx < len(cmd); idx++ {
		if err := CheckSingleCommand(cmd[idx], tee[idx]); err != nil {
			return err
		}
	}
	return nil
}

// SimpleCommandCollector implements [shellx.Dependencies] and
// tracks all the commands that have been run.
type SimpleCommandCollector struct {
	Commands []*execabs.Cmd
}

var _ shellx.Dependencies = &SimpleCommandCollector{}

// CmdOutput implements shellx.Dependencies
func (cc *SimpleCommandCollector) CmdOutput(c *execabs.Cmd) ([]byte, error) {
	cc.Commands = append(cc.Commands, c)
	return nil, nil // a command that does not fail and does not emit any output
}

// CmdRun implements shellx.Dependencies
func (cc *SimpleCommandCollector) CmdRun(c *execabs.Cmd) error {
	cc.Commands = append(cc.Commands, c)
	return nil
}

// LookPath implements shellx.Dependencies
func (cc *SimpleCommandCollector) LookPath(file string) (string, error) {
	return file, nil
}

// CanonicalGolangVersion is the canonical Go version used in tests.
const CanonicalGolangVersion = "1.14.17"

// Constants describing the dependent functions we can call when building.
const (
	TagAbsoluteCurDir              = "absoluteCurDir"
	TagAndroidNDKCheck             = "androidNDK"
	TagAndroidSDKCheck             = "androidSDK"
	TagGOPATH                      = "GOPATH"
	TagGolangCheck                 = "golangCheck"
	TagLinuxReadGOVERSION          = "linuxReadGOVERSION"
	TagLinuxWriteDockerfile        = "linuxWriteDockerfile"
	TagMustChdir                   = "mustChdir"
	TagPsiphonFilesExist           = "psiphonFilesExist"
	TagPsiphonMaybeCopyConfigFiles = "maybeCopyPsiphonFiles"
	TagVerifySHA256                = "verifySHA256"
	TagWindowsMingwCheck           = "windowsMingwCheck"
)

// DependenciesCallCounter allows to counter how many times the
// build dependencies have been called in a run.
type DependenciesCallCounter struct {
	Counter    map[string]int
	HasPsiphon bool
}

var _ buildtoolmodel.Dependencies = &DependenciesCallCounter{}

// CanonicalNDKVersion is the canonical NDK version used in tests.
const CanonicalNDKVersion = "25.1.7654321"

// AbsoluteCurDir implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) AbsoluteCurDir() string {
	cc.increment(TagAbsoluteCurDir)
	return runtimex.Try1(filepath.Abs("../../../")) // pretend we're in the real topdir
}

// AndroidNDKCheck implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) AndroidNDKCheck(androidHome string) string {
	cc.increment(TagAndroidNDKCheck)
	return filepath.Join(androidHome, "ndk", CanonicalNDKVersion)
}

// AndroidSDKCheck implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) AndroidSDKCheck() string {
	cc.increment(TagAndroidSDKCheck)
	return filepath.Join("", "Android", "sdk") // fake location
}

// GOPATH implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) GOPATH() string {
	cc.increment(TagGOPATH)
	return "/go/gopath" // fake location
}

// golangCheck implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) GolangCheck() {
	cc.increment(TagGolangCheck)
}

// linuxReadGOVERSION implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) LinuxReadGOVERSION(filename string) []byte {
	cc.increment(TagLinuxReadGOVERSION)
	v := append([]byte(CanonicalGolangVersion), '\n')
	return v
}

// linuxWriteDockerfile implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) LinuxWriteDockerfile(
	filename string, content []byte, mode fs.FileMode) {
	cc.increment(TagLinuxWriteDockerfile)
}

// MustChdir implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) MustChdir(dirname string) func() {
	cc.increment(TagMustChdir)
	return func() {} // nothing
}

// psiphonFilesExist implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) PsiphonFilesExist() bool {
	cc.increment(TagPsiphonFilesExist)
	return cc.HasPsiphon
}

// psiphonMaybeCopyConfigFiles implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) PsiphonMaybeCopyConfigFiles() {
	cc.increment(TagPsiphonMaybeCopyConfigFiles)
}

// VerifySHA256 implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) VerifySHA256(expectedSHA256 string, tarball string) {
	cc.increment(TagVerifySHA256)
}

// windowsMingwCheck implements buildtoolmodel.Dependencies
func (cc *DependenciesCallCounter) WindowsMingwCheck() {
	cc.increment(TagWindowsMingwCheck)
}

// increment increments the given counter
func (cc *DependenciesCallCounter) increment(name string) {
	if cc.Counter == nil {
		cc.Counter = make(map[string]int)
	}
	cc.Counter[name]++
}
