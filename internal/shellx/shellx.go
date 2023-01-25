// Package shellx helps to write shell-like Go code.
package shellx

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/google/shlex"
	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"golang.org/x/sys/execabs"
)

// Dependencies is the library on which this package depends.
type Dependencies interface {
	// CmdOutput is equivalent to calling c.Output.
	CmdOutput(c *execabs.Cmd) ([]byte, error)

	// CmdRun is equivalent to calling c.Run.
	CmdRun(c *execabs.Cmd) error

	// LookPath is equivalent to calling execabs.LookPath.
	LookPath(file string) (string, error)
}

// Library contains the default dependencies.
var Library Dependencies = &StdlibDependencies{}

// StdlibDependencies contains the stdlib implementation of the [Dependencies].
type StdlibDependencies struct{}

// CmdOutput implements [Dependencies].
func (*StdlibDependencies) CmdOutput(c *execabs.Cmd) ([]byte, error) {
	return c.Output()
}

// CmdRun implements [Dependencies].
func (*StdlibDependencies) CmdRun(c *execabs.Cmd) error {
	return c.Run()
}

// LookPath implements [Dependencies].
func (*StdlibDependencies) LookPath(file string) (string, error) {
	return execabs.LookPath(file)
}

// Envp is the environment in which we execute commands.
type Envp struct {
	// V contains the OPTIONAL environment variables to add to the current
	// environment when we're executing commands.
	V []string
}

// Append appends an environment variable to the environment.
func (e *Envp) Append(key, value string) {
	e.V = append(e.V, fmt.Sprintf("%s=%s", key, value))
}

// Argv contains the complete argv.
type Argv struct {
	// P is the MANDATORY program to execute.
	P string

	// V contains the OPTIONAL arguments.
	V []string
}

// NewArgv creates a new [Argv] from the given command and arguments.
func NewArgv(command string, args ...string) (*Argv, error) {
	fullpath, err := Library.LookPath(command) // allows mocking
	if err != nil {
		return nil, err
	}
	argv := &Argv{
		P: fullpath,
		V: args,
	}
	return argv, nil
}

// ParseCommandLine creates an instance of [Argv] from the given command line.
func ParseCommandLine(cmdline string) (*Argv, error) {
	args, err := shlex.Split(cmdline)
	if err != nil {
		return nil, err
	}
	if len(args) < 1 {
		return nil, ErrNoCommandToExecute
	}
	return NewArgv(args[0], args[1:]...)
}

// Append appends arguments to the command line.
func (a *Argv) Append(args ...string) {
	a.V = append(a.V, args...)
}

const (
	// FlagShowStdoutStderr enables connecting the child's stdout and stderr
	// to the current program's stdout and stderr.
	FlagShowStdoutStderr = 1 << iota
)

// Config contains config for executing programs.
type Config struct {
	// Logger is the OPTIONAL logger to use.
	Logger model.Logger

	// Flags contains OPTIONAL binary flags to configure the program.
	Flags int64
}

// cmd creates a new [execabs.Cmd] instance.
func cmd(config *Config, argv *Argv, envp *Envp) *execabs.Cmd {
	// Implementation note: since Go 1.19 we don't need to use the execabs
	// package anymore. See <https://tip.golang.org/doc/go1.19>.
	cmd := execabs.Command(argv.P, argv.V...)
	cmd.Env = os.Environ()
	for _, entry := range envp.V {
		if config.Logger != nil {
			config.Logger.Infof("+ export %s", entry)
		}
		cmd.Env = append(cmd.Env, entry)
	}
	if config.Logger != nil {
		cmdline := quotedCommandLine(argv.P, argv.V...)
		config.Logger.Infof("+ %s", cmdline)
	}
	return cmd
}

// OutputEx implements [Output] and [OutputQuiet].
func OutputEx(config *Config, argv *Argv, envp *Envp) ([]byte, error) {
	cmd := cmd(config, argv, envp)
	if (config.Flags & FlagShowStdoutStderr) != 0 {
		// note: cmd.Output wants the stdout to be nil
		cmd.Stderr = os.Stderr
	}
	return Library.CmdOutput(cmd) // allows mocking
}

// output is the common implementation of [Output] and [OutputQuiet].
func output(logger model.Logger, flags int64, command string, args ...string) ([]byte, error) {
	argv, err := NewArgv(command, args...)
	if err != nil {
		return nil, err
	}
	envp := &Envp{}
	config := &Config{
		Logger: logger,
		Flags:  flags,
	}
	return OutputEx(config, argv, envp)
}

// OutputQuiet is like [RunQuiet] except that, in case of success, it captures
// the standard output and returns it to the caller.
func OutputQuiet(command string, args ...string) ([]byte, error) {
	return output(nil, 0, command, args...)
}

// Output is like [OutputQuiet] except that it logs the command to be executed
// and the environment variables specific to this command.
func Output(logger model.Logger, command string, args ...string) ([]byte, error) {
	return output(logger, FlagShowStdoutStderr, command, args...)
}

// RunEx implements [Run] and [RunQuiet].
func RunEx(config *Config, argv *Argv, envp *Envp) error {
	cmd := cmd(config, argv, envp)
	if config.Flags&FlagShowStdoutStderr != 0 {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return Library.CmdRun(cmd) // allows mocking
}

// run is the common implementation of [Run] and [RunQuiet].
func run(logger model.Logger, flags int64, command string, args ...string) error {
	argv, err := NewArgv(command, args...)
	if err != nil {
		return err
	}
	envp := &Envp{}
	config := &Config{
		Logger: logger,
		Flags:  flags,
	}
	return RunEx(config, argv, envp)
}

// RunQuiet runs the given command without emitting any output and
// using the environment variables in the current [Envp].
func RunQuiet(command string, args ...string) error {
	return run(nil, 0, command, args...)
}

// Run is like [RunQuiet] except that it also logs the command to
// exec, the environment variables specific to this command, the text
// logged to stdout and stderr.
func Run(logger model.Logger, command string, args ...string) error {
	return run(logger, FlagShowStdoutStderr, command, args...)
}

// runCommandLine is the common implementation of
// [RunCommandLineQuiet] and [RunCommandLine].
func runCommandLine(logger model.Logger, flags int64, cmdline string) error {
	argv, err := ParseCommandLine(cmdline)
	if err != nil {
		return err
	}
	envp := &Envp{}
	config := &Config{
		Logger: logger,
		Flags:  flags,
	}
	return RunEx(config, argv, envp)
}

// RunCommandLineQuiet is like [RunQuiet] but takes a command line as argument.
func RunCommandLineQuiet(cmdline string) error {
	return runCommandLine(nil, 0, cmdline)
}

// RunCommandLine is like [RunCommandLineQuiet] but logs the command to
// execute as well as the command-specific environment variables.
func RunCommandLine(logger model.Logger, cmdline string) error {
	return runCommandLine(logger, FlagShowStdoutStderr, cmdline)
}

// outputCommandLine is the common implementation
// of [OutputCommandLineQuiet] and [OutputCommandLine].
func outputCommandLine(logger model.Logger, flags int64, cmdline string) ([]byte, error) {
	argv, err := ParseCommandLine(cmdline)
	if err != nil {
		return nil, err
	}
	envp := &Envp{}
	config := &Config{
		Logger: logger,
		Flags:  flags,
	}
	return OutputEx(config, argv, envp)
}

// OutputCommandLineQuiet is like [OutputQuiet] but takes a command line as argument.
func OutputCommandLineQuiet(cmdline string) ([]byte, error) {
	return outputCommandLine(nil, 0, cmdline)
}

// OutputCommandLine is like OutputCommandLineQuiet but logs the command to
// execute as well as the command-specific environment variables.
func OutputCommandLine(logger model.Logger, cmdline string) ([]byte, error) {
	return outputCommandLine(logger, FlagShowStdoutStderr, cmdline)
}

// ErrNoCommandToExecute means that the command line is empty.
var ErrNoCommandToExecute = errors.New("shellx: no command to execute")

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

// fsxOpenFile is the function to open a file for reading.
var fsxOpenFile = fsx.OpenFile

// osOpenFile is the generic function to open a file.
var osOpenFile = os.OpenFile

// netxliteCopyContext is the generic function to copy content.
var netxliteCopyContext = netxlite.CopyContext

// CopyFile copies [source] to [dest].
func CopyFile(source, dest string, perms fs.FileMode) error {
	sourcefp, err := fsxOpenFile(source)
	if err != nil {
		return err
	}
	defer sourcefp.Close()
	destfp, err := osOpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perms)
	if err != nil {
		return err
	}
	if _, err := netxliteCopyContext(context.Background(), destfp, sourcefp); err != nil {
		destfp.Close()
		return err
	}
	return destfp.Close()
}
