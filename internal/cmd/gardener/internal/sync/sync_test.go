package sync_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/sync"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
	"golang.org/x/sys/execabs"
)

// commandCollector implements [shellx.Dependencies] and
// tracks all the commands that have been run.
type commandCollector struct {
	Commands []*execabs.Cmd
}

var _ shellx.Dependencies = &commandCollector{}

// CmdOutput implements shellx.Dependencies
func (cc *commandCollector) CmdOutput(c *execabs.Cmd) ([]byte, error) {
	cc.Commands = append(cc.Commands, c)
	return nil, nil // a command that does not fail and does not emit any output
}

// CmdRun implements shellx.Dependencies
func (cc *commandCollector) CmdRun(c *execabs.Cmd) error {
	cc.Commands = append(cc.Commands, c)
	return nil
}

// LookPath implements shellx.Dependencies
func (cc *commandCollector) LookPath(file string) (string, error) {
	return file, nil
}

// Chdir is called when we attempt to change directory
func (cc *commandCollector) Chdir(dir string) error {
	cmd := &execabs.Cmd{
		Path: "cd",
		Args: []string{
			"cd", dir,
		},
	}
	cc.Commands = append(cc.Commands, cmd)
	return nil
}

// Getwd returns the current working directory
func (cc *commandCollector) Getwd() (string, error) {
	return "/workdir", nil
}

// TimeNow returns the current time
func (cc *commandCollector) TimeNow() time.Time {
	// This is a not-so-obvious reference to the following pull request
	// https://github.com/measurement-kit/measurement-kit/pull/1924
	return time.Date(2023, time.March, 15, 11, 43, 00, 00, time.UTC)
}

func TestWorkingAsIntended(t *testing.T) {
	// make sure we collect the commands we _would_ execute
	cc := &commandCollector{}

	// create the subcommand instance
	repodir := filepath.Join("testdata", "repo")
	dnsreportfile := filepath.Join("testdata", "dnsreport.sqlite3")
	sc := &sync.Subcommand{
		DNSReportDatabase: dnsreportfile,
		RepositoryDir:     repodir,
		OsChdir:           cc.Chdir,
		OsGetwd:           cc.Getwd,
		TimeNow:           cc.TimeNow,
	}

	// run the subcommand with custom shellx dependencies
	shellxtesting.WithCustomLibrary(cc, func() {
		sc.Main()
	})

	// expectations for commands
	expect := []string{
		fmt.Sprintf("rm -rf %s", repodir),
		fmt.Sprintf("rm -f %s", dnsreportfile),
		fmt.Sprintf("git clone https://github.com/citizenlab/test-lists %s", repodir),
		fmt.Sprintf("cd %s", repodir),
		"git checkout -b gardener_20230315T114300Z",
		"cd /workdir",
	}

	// make sure the number of commands is consistent with expectations
	if len(cc.Commands) != len(expect) {
		t.Fatal("expected", len(expect), "commands, got", len(cc.Commands))
	}

	// verify expectations for commands
	for idx := 0; idx < len(expect); idx++ {
		if !strings.HasSuffix(cc.Commands[idx].String(), expect[idx]) {
			t.Fatal("expected suffix", expect[idx], "got", cc.Commands[idx].String())
		}
	}
}
