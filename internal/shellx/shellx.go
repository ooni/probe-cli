// Package shellx runs external commands.
package shellx

import (
	"errors"
	"os"
	"strings"

	"github.com/google/shlex"
	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/sys/execabs"
)

// runconfig is the configuration for run.
type runconfig struct {
	// args contains the command line arguments.
	args []string

	// command is the command to execute.
	command string

	// loginfof is the logging function.
	loginfof func(format string, v ...interface{})

	// stdout is the standard output.
	stdout *os.File

	// stderr is the standard error.
	stderr *os.File
}

// run is the internal function for running commands.
func run(config runconfig) error {
	config.loginfof(
		"exec: %s %s", config.command, strings.Join(config.args, " "))
	// implementation note: here we're using execabs because
	// of https://blog.golang.org/path-security.
	cmd := execabs.Command(config.command, config.args...)
	cmd.Stdout = config.stdout
	cmd.Stderr = config.stderr
	err := cmd.Run()
	config.loginfof("exec result: %+v", err)
	return err
}

// Run executes the specified command with the specified args.
func Run(logger model.InfoLogger, name string, arg ...string) error {
	return run(runconfig{
		args:     arg,
		command:  name,
		loginfof: logger.Infof,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
	})
}

// quietInfof is an infof function that does nothing.
func quietInfof(format string, v ...interface{}) {}

// RunQuiet is like Run but it does not emit any output.
func RunQuiet(name string, arg ...string) error {
	return run(runconfig{
		args:     arg,
		command:  name,
		loginfof: quietInfof,
		stdout:   nil,
		stderr:   nil,
	})
}

// ErrNoCommandToExecute means that the command line is empty.
var ErrNoCommandToExecute = errors.New("shellx: no command to execute")

// RunCommandline executes the given command line.
func RunCommandline(logger model.InfoLogger, cmdline string) error {
	args, err := shlex.Split(cmdline)
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return ErrNoCommandToExecute
	}
	return Run(logger, args[0], args[1:]...)
}
