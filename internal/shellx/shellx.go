// Package shellx runs external commands.
package shellx

import (
	"errors"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/google/shlex"
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

// Run executes the specified command with the specified args
func Run(name string, arg ...string) error {
	return run(runconfig{
		args:     arg,
		command:  name,
		loginfof: log.Log.Infof,
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

// RunCommandline is like Run but its only argument is a command
// line that will be splitted using the google/shlex package.
func RunCommandline(cmdline string) error {
	args, err := shlex.Split(cmdline)
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return errors.New("shellx: no command to execute")
	}
	return Run(args[0], args[1:]...)
}
