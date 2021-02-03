// Package shellx contains utilities to run external commands.
package shellx

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/apex/log"
	"github.com/google/shlex"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

type runconfig struct {
	args     []string
	loginfof func(format string, v ...interface{})
	name     string
	stdout   *os.File
	stderr   *os.File
}

func run(config runconfig) error {
	config.loginfof("exec: %s %s", config.name, strings.Join(config.args, " "))
	cmd := exec.Command(config.name, config.args...)
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
		loginfof: log.Log.Infof,
		name:     name,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
	})
}

// RunQuiet is like Run but it does not emit any output.
func RunQuiet(name string, arg ...string) error {
	return run(runconfig{
		args:     arg,
		loginfof: model.DiscardLogger.Infof,
		name:     name,
		stdout:   nil,
		stderr:   nil,
	})
}

// RunCommandline is like Run but its only argument is a command
// line that will be splitted using the google/shlex package
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
