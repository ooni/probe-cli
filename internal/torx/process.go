package torx

//
// process.go - executing tor as an external process.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/sys/execabs"
)

// ExecDeps contains dependencies for [Exec].
type ExecDeps interface {
	// Getenv must behave like os.Getenv.
	Getenv(key string) string

	// LookPath must behave like execabs.LookPath.
	LookPath(file string) (string, error)

	// StartCmd invokes the cmd.Start.
	StartCmd(cmd *execabs.Cmd) error
}

// execDepsStdlib implements [ExecDeps] using the standard library.
type execDepsStdlib struct{}

var _ ExecDeps = execDepsStdlib{}

// Getenv implements Deps.
func (execDepsStdlib) Getenv(key string) string {
	return os.Getenv(key)
}

// LookPath implements Deps.
func (execDepsStdlib) LookPath(file string) (string, error) {
	return execabs.LookPath(file)
}

// StartCmd implements Deps.
func (execDepsStdlib) StartCmd(cmd *exec.Cmd) error {
	return cmd.Start()
}

// ProcessState contains information about a process as
// reported by the [Process] Wait method.
type ProcessState interface {
	ExitCode() int
}

// Process is a running tor process.
type Process interface {
	// DialControl attempts to dial a control connection
	// with the tor process and returns the results.
	DialControl(ctx context.Context) (*ControlConn, error)

	// Kill kills the process immediately (this corresponds to
	// sending a SIGKILL in POSIX compliant systems).
	//
	// If possible, you should try to shutdown tor gracefully
	// using TAKEOWNERSHIP with the control conn.
	Kill() error

	// Wait waits for the tor process to terminate and returns
	// information about how the process exited.
	Wait() (ProcessState, error)
}

// ExecOption is an option for [Exec].
type ExecOption func(config *execOptions)

// execOptions contains all the options for [Exec].
type execOptions struct {
	// deps contains the dependencies.
	deps ExecDeps

	// torBinary overrides the binary to exec.
	torBinary string

	// torExtraArgs provides extra command line arguments.
	torExtraArgs []string
}

// ExecOptionDeps sets the [ExecDeps].
func ExecOptionDeps(deps ExecDeps) ExecOption {
	return func(config *execOptions) {
		config.deps = deps
	}
}

// ExecOptionTorBinary sets the tor binary to execute.
func ExecOptionTorBinary(path string) ExecOption {
	return func(config *execOptions) {
		config.torBinary = path
	}
}

// ExecOptionAppendExtraArgs appends extra args to the command line.
func ExecOptionAppendExtraArgs(args ...string) ExecOption {
	return func(config *execOptions) {
		config.torExtraArgs = append(config.torExtraArgs, args...)
	}
}

// ExecOptionUsePluggableTransport appends arguments to the command line to
// use the pluggable transport described by the given [PTInfo].
func ExecOptionUsePluggableTransport(info PTInfo) ExecOption {
	return func(config *execOptions) {
		extraArgs := []string{
			"UseBridges", "1",
			"ClientTransportPlugin", info.AsClientTransportPluginArgument(),
			"Bridge", info.AsBridgeArgument(),
		}
		config.torExtraArgs = append(config.torExtraArgs, extraArgs...)
	}
}

// Exec executes the tor binary and returns the corresponding process.
//
// We use the following algorithm to decide which binary to execute:
//
// 1. if Options.TorBinary is specified, we use it;
//
// 2. if the OONI_TOR_BINARY environment variable exists and is
// not empty, we use its value as the tor binary;
//
// 3. otherwise we use "tor".
//
// In the first and the third case, we use execabs.LookPath to make
// sure we're going to execute a binary belonging to the PATH.
//
// We create the datadir if it does not exist. However, we do not
// remove the datadir unless the specific option is set.
func Exec(datadir *DataDirState, logger model.Logger, options ...ExecOption) (Process, error) {
	// init the config
	config := &execOptions{
		deps:         &execDepsStdlib{},
		torBinary:    "",
		torExtraArgs: []string{},
	}
	for _, option := range options {
		option(config)
	}

	// figure out the name of the process to execute
	torBinaryPath, err := getTorBinaryPath(config.deps, config)
	if err != nil {
		return nil, err
	}

	// fill the default command line arguments
	defaultArgs := []string{
		torBinaryPath,
		"-f", datadir.TorRcFile,
		"ControlPort", "auto",
		"ControlPortWriteToFile", datadir.ControlPortFile,
		"CookieAuthentication", "1",
		"CookieAuthFile", datadir.CookieAuthFile,
		"DataDirectory", datadir.DirPath,
		"DisableNetwork", "1",
		"SocksPort", "auto",
	}

	// create the command to execute
	cmd := &execabs.Cmd{
		Path:   torBinaryPath,
		Args:   append(defaultArgs, config.torExtraArgs...),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	logger.Infof("torx: process: exec: %s", cmd.String())

	// execute the tor binary
	if err := config.deps.StartCmd(cmd); err != nil {
		return nil, err
	}

	// create and return a process instance
	proc := &process{
		logger: logger,
		p:      cmd.Process,
		state:  datadir,
	}
	return proc, nil
}

// process implements [Process]
type process struct {
	// logger is the logger to use.
	logger model.Logger

	// p is the running process.
	p *os.Process

	// state is the data dir state.
	state *DataDirState
}

var _ Process = &process{}

// ErrCannotReachControlPort indicates we cannot reach the control port.
var ErrCannotReachControlPort = errors.New("tor: cannot reach control port")

// DialControl implements Process.
func (p *process) DialControl(ctx context.Context) (*ControlConn, error) {
	return dialControl(ctx, p.state, p.logger)
}

// dialControl is the common function for dialing a control connection.
func dialControl(ctx context.Context, dataDir *DataDirState, logger model.Logger) (*ControlConn, error) {
	dialer := &net.Dialer{}
	const waitDelay = 200 * time.Millisecond

	for i := 0; i < 10; i++ {
		endpoint, err := readControlPortFile(dataDir)
		if err != nil {
			select {
			case <-time.After(waitDelay):
				// all good
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}

		conn, err := dialer.DialContext(ctx, "tcp", endpoint)
		if err != nil {
			select {
			case <-time.After(waitDelay):
				// all good
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}

		return NewControlConn(conn, logger), nil
	}

	return nil, ErrCannotReachControlPort
}

// errControlMissingPortPrefix means we're missing the PORT= prefix.
var errControlMissingPortPrefix = errors.New("tor: control: missing the PORT prefix")

func readControlPortFile(datadir *DataDirState) (string, error) {
	data, err := os.ReadFile(datadir.ControlPortFile)
	if err != nil {
		return "", err
	}
	if !bytes.HasPrefix(data, []byte("PORT=")) {
		return "", errControlMissingPortPrefix
	}
	endpoint := strings.TrimRight(string(data[5:]), "\r\n")
	return endpoint, nil
}

// Kill implements Process.
func (p *process) Kill() error {
	return p.p.Kill()
}

// Wait implements Process.
func (p *process) Wait() (ProcessState, error) {
	return p.p.Wait()
}

func getTorBinaryPath(deps ExecDeps, config *execOptions) (string, error) {
	// ooniTorBinaryEnv is the name of the environment variable
	// we're using to get the path to the tor binary when we are
	// being run by the ooni/probe-desktop application.
	const ooniTorBinaryEnv = "OONI_TOR_BINARY"

	// 1
	if config.torBinary != "" {
		return deps.LookPath(config.torBinary)
	}

	// 2
	if binary := deps.Getenv(ooniTorBinaryEnv); binary != "" {
		return binary, nil
	}

	// 3
	return deps.LookPath("tor")
}
