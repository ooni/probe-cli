// Package shellx runs external commands.
package shellx

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/shlex"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// Dependencies is the library on which this package depends.
type Dependencies interface {
	// CmdOutput is equivalent to calling c.Output.
	CmdOutput(c *exec.Cmd) ([]byte, error)

	// CmdRun is equivalent to calling c.Run.
	CmdRun(c *exec.Cmd) error

	// LookPath is equivalent to calling exec.LookPath.
	LookPath(file string) (string, error)
}

// Library contains the default dependencies. You will want to change
// this variable when writing tests.
var Library Dependencies = &StdlibDependencies{}

// StdlibDependencies contains the stdlib implementation of the [Dependencies].
type StdlibDependencies struct{}

// CmdOutput implements Dependencies
func (*StdlibDependencies) CmdOutput(c *exec.Cmd) ([]byte, error) {
	return c.Output()
}

// CmdRun implements Dependencies
func (*StdlibDependencies) CmdRun(c *exec.Cmd) error {
	return c.Run()
}

// LookPath implements Dependencies
func (*StdlibDependencies) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// Env is the environment in which we execute commands.
type Env struct {
	// Vars contains the environment variables to add to the current
	// environment when we're executing commands.
	Vars []string
}

// Append appends an environment variable to the environment.
func (e *Env) Append(key, value string) {
	e.Vars = append(e.Vars, fmt.Sprintf("%s=%s", key, value))
}

// OutputQuiet is like RunQuiet except that, in case of success, it captures
// the standard output and returns it to the caller.
func (e *Env) OutputQuiet(command string, args ...string) ([]byte, error) {
	return e.Output(nil, command, args...)
}

// Output is like OutputQuiet except that it logs the command to be executed
// and the environment variables specific to this command.
func (e *Env) Output(logger model.Logger, command string, args ...string) ([]byte, error) {
	cmd, err := e.cmd(logger, command, args...)
	if err != nil {
		return nil, err
	}
	if logger != nil {
		// note: cmd.Output wants the stdout to be nil
		cmd.Stderr = os.Stderr
	}
	return Library.CmdOutput(cmd) // allows mocking
}

// RunQuiet runs the given command without emitting any output and
// using the environment variables in the current [Environ].
func (e *Env) RunQuiet(command string, args ...string) error {
	return Run(nil, command, args...)
}

// Run is like RunQuiet except that it also logs the command to be
// executed, the environment variables specific to this command, the
// text logged to stdout and stderr.
func (e *Env) Run(logger model.Logger, command string, args ...string) error {
	cmd, err := e.cmd(logger, command, args...)
	if err != nil {
		return err
	}
	if logger != nil {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return Library.CmdRun(cmd) // allows mocking
}

// RunCommandLineQuiet is like RunQuiet but takes a command line as argument.
func (e *Env) RunCommandLineQuiet(cmdline string) error {
	return e.RunCommandLine(nil, cmdline)
}

// RunCommandLine is like RunCommandLineQuiet but logs the command to
// execute as well as the command-specific environment variables.
func (e *Env) RunCommandLine(logger model.Logger, cmdline string) error {
	args, err := shlex.Split(cmdline)
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return ErrNoCommandToExecute
	}
	return e.Run(logger, args[0], args[1:]...)
}

// OutputCommandLineQuiet is like OutputQuiet but takes a command line as argument.
func (e *Env) OutputCommandLineQuiet(cmdline string) ([]byte, error) {
	return e.OutputCommandLine(nil, cmdline)
}

// OutputCommandLine is like OutputCommandLineQuiet but logs the command to
// execute as well as the command-specific environment variables.
func (e *Env) OutputCommandLine(logger model.Logger, cmdline string) ([]byte, error) {
	args, err := shlex.Split(cmdline)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, ErrNoCommandToExecute
	}
	return e.Output(logger, args[0], args[1:]...)
}

// cmd is an internal factory for creating a new command.
func (e *Env) cmd(logger model.Logger, command string, args ...string) (*exec.Cmd, error) {
	fullpath, err := Library.LookPath(command) // allows mocking
	if err != nil {
		return nil, err
	}
	// Implementation note: since Go 1.19 we don't need to use the execabs
	// package anymore. See <https://tip.golang.org/doc/go1.19>.
	cmd := exec.Command(fullpath, args...)
	cmd.Env = os.Environ()
	for _, entry := range e.Vars {
		if logger != nil {
			logger.Infof("+ export %s", entry)
		}
		cmd.Env = append(cmd.Env, entry)
	}
	if logger != nil {
		cmdline := quotedCommandLine(fullpath, args...)
		logger.Infof("+ %s", cmdline)
	}
	return cmd, nil
}

// Run calls [Env.Run] using an empty [Env].
func Run(logger model.Logger, program string, args ...string) error {
	return (&Env{}).Run(logger, program, args...)
}

// RunQuiet calls [Env.RunQuiet] using an empty [Env].
func RunQuiet(program string, args ...string) error {
	return (&Env{}).RunQuiet(program, args...)
}

// ErrNoCommandToExecute means that the command line is empty.
var ErrNoCommandToExecute = errors.New("shellx: no command to execute")

// RunCommandLine calls [Env.RunCommandLine] using an empty [Env].
func RunCommandLine(logger model.Logger, cmdline string) error {
	return (&Env{}).RunCommandLine(logger, cmdline)
}

// RunCommandLineQuiet calls [Env.RunCommandLineQuiet] using an empty [Env].
func RunCommandLineQuiet(cmdline string) error {
	return (&Env{}).RunCommandLineQuiet(cmdline)
}

// quotedCommandLine returns a quoted command line.
func quotedCommandLine(command string, args ...string) string {
	v := []string{}
	v = append(v, maybeQuoteArg(command))
	for _, a := range args {
		v = append(v, maybeQuoteArg(a))
	}
	return strings.Join(v, " ")
}

// maybeQuoteArg quotes a command line argument if needed.
func maybeQuoteArg(a string) string {
	if strings.Contains(a, "\"") {
		a = strings.ReplaceAll(a, "\"", "\\\"")
	}
	if strings.Contains(a, " ") {
		a = "\"" + a + "\""
	}
	return a
}
