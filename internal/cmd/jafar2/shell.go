package main

import (
	"github.com/apex/log"
	"golang.org/x/sys/execabs"
)

// Shell executes commands.
type Shell interface {
	// Run runs the given command.
	Run(cmd *execabs.Cmd) error

	// MustRun is like Run but exits on error.
	MustRun(cmd *execabs.Cmd)

	// MustCaptureOutput caputures the shell output.
	//
	// Remark: make sure cmd.Stdout is nil before calling
	// this method, otherwise the code will fail.
	MustCaptureOutput(cmd *execabs.Cmd) []byte
}

// LinuxShell is a shell that works on Linux.
type LinuxShell struct{}

var _ Shell = &LinuxShell{}

// NewLinuxShell creates a new LinuxShell instance.
func NewLinuxShell() *LinuxShell {
	return &LinuxShell{}
}

// Run implements Shell.Run.
func (sh *LinuxShell) Run(cmd *execabs.Cmd) error {
	log.Debugf("exec: %s...", cmd.String())
	err := cmd.Run()
	log.Debugf("exec: %s... %s", cmd.String(), sh.errorAsString(err))
	return err
}

// MustRun implements Shell.MustRun.
func (sh *LinuxShell) MustRun(cmd *execabs.Cmd) {
	err := sh.Run(cmd)
	FatalOnError(err, "cannot run command")
}

// MustCaptureOutput implements Shell.MustCaptureOutput.
func (sh *LinuxShell) MustCaptureOutput(cmd *execabs.Cmd) []byte {
	out, err := cmd.Output()
	FatalOnError(err, "cannot capture command's output")
	return out
}

func (sh *LinuxShell) errorAsString(err error) string {
	if err != nil {
		return err.Error()
	}
	return "<nil>"
}
