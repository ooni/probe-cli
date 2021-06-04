// Package shellx runs external commands.
package shellx

import (
	"errors"
	"fmt"
	"os"
	"strings"

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
		"exec: %s %s\n", config.command, strings.Join(config.args, " "))
	// implementation note: here we're using execabs because
	// of https://blog.golang.org/path-security.
	cmd := execabs.Command(config.command, config.args...)
	cmd.Stdout = config.stdout
	cmd.Stderr = config.stderr
	err := cmd.Run()
	config.loginfof("exec result: %+v\n", err)
	return err
}

// noisyInfof is an infof function printing on the stderr.
func noisyInfof(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

// Run executes the specified command with the specified args.
func Run(name string, arg ...string) error {
	return run(runconfig{
		args:     arg,
		command:  name,
		loginfof: noisyInfof,
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
func RunCommandline(cmdline string) error {
	args, err := shlex.Split(cmdline)
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return ErrNoCommandToExecute
	}
	return Run(args[0], args[1:]...)
}
