package main

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// replaceDeps contains dependencies for [replaceRunningInstance].
type replaceDeps struct {
	// CopyFile is MANDATORY and MUST behave like [shellx.CopyFile].
	CopyFile func(source string, dest string, perms fs.FileMode) error

	// Run is MANDATORY and MUST behave like [shellx.Run].
	Run func(logger model.Logger, command string, args ...string) error
}

// replaceDryRun is an internal flag used for testing.
var replaceDryRun bool

// newReplaceDeps creates a fully initialized instance of [replaceDeps].
func newReplaceDeps() *replaceDeps {
	deps := &replaceDeps{
		CopyFile: shellx.CopyFile,
		Run:      shellx.Run,
	}
	if replaceDryRun {
		deps.CopyFile = func(source, dest string, perms fs.FileMode) error {
			return nil
		}
		deps.Run = func(logger model.Logger, command string, args ...string) error {
			return nil
		}
	}
	return deps
}

// replaceRunningInstance executes the command to replace
// a running instance of oohelperd with this instance.
func replaceRunningInstance(deps *replaceDeps) {
	// stop the running instance.
	runtimex.Try0(deps.Run(log.Log, "systemctl", "stop", "oohelperd.service"))

	// copy oohelperd to the destination path.
	executable := runtimex.Try1(filepath.Abs(runtimex.Try1(os.Executable())))
	destpath := string(filepath.Separator) + filepath.Join("usr", "bin", "oohelperd")
	log.Infof("+ cp %s %s", executable, destpath)
	runtimex.Try0(deps.CopyFile(executable, destpath, 0755))

	// restart the running instance.
	runtimex.Try0(deps.Run(log.Log, "systemctl", "start", "oohelperd.service"))
}
