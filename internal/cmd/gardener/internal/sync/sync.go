// Package sync implements the sync subcommand.
package sync

import (
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// testListsRepo is the citizenlab/test-lists repository URL.
const testListsRepo = "https://github.com/citizenlab/test-lists"

// Subcommand is the sync subcommand. The zero value is invalid; please, make
// sure you initialize all the fields marked as MANDATORY.
type Subcommand struct {
	// RepositoryDir is the MANDATORY directory where to clone the test lists repository.
	RepositoryDir string

	// OsChdir is the MANDATORY function to change directory.
	OsChdir func(dir string) error

	// OsGetwd is the MANDATORY function to get the current working dir.
	OsGetwd func() (string, error)

	// TimeNow is the MANDATORY function to get the current time.
	TimeNow func() time.Time
}

// Main is the main function run by the sync subcommand. This function calls
// [runtimex.PanicOnError] in case of failure.
func (s *Subcommand) Main() {
	// possibly remove a previous working copy
	runtimex.Try0(shellx.Run(log.Log, "rm", "-rf", s.RepositoryDir))

	// clone a new working copy
	runtimex.Try0(shellx.Run(log.Log, "git", "clone", testListsRepo, s.RepositoryDir))

	// get the current working directory
	cwd := runtimex.Try1(s.OsGetwd())

	// enter into the git repository directory
	log.Infof("+ chdir %s", s.RepositoryDir)
	runtimex.Try0(s.OsChdir(s.RepositoryDir))

	// create a unique branch name for this session
	branchName := fmt.Sprintf("gardener_%s", s.TimeNow().UTC().Format("20060102T150405Z"))

	// checkout a working branch to avoid someone making a mistake
	// and pushing directly on the main branch
	runtimex.Try0(shellx.Run(log.Log, "git", "checkout", "-b", branchName))

	// return to the previous working directory
	log.Infof("+ chdir %s", cwd)
	runtimex.Try0(s.OsChdir(cwd))
}
