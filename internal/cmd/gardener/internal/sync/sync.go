// Package sync contains the subcommand to synchronize the test lists.
package sync

import (
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
}

// Main is the main function run by the sync subcommand.
func (sc *Subcommand) Main() {
	// possibly remove a previous working copy
	runtimex.Try0(shellx.Run(log.Log, "rm", "-rf", sc.RepositoryDir))

	// clone a new working copy
	runtimex.Try0(shellx.Run(log.Log, "git", "clone", testListsRepo, sc.RepositoryDir))
}
