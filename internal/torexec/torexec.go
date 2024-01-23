// Package torexec contains code to execute tor.
package torexec

//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tordatadir"
	"golang.org/x/sys/execabs"
)

// Deps contains this package runtime dependencies.
type Deps interface {
	// We embed tordatadir deps
	tordatadir.Deps

	// Getenv must behave like os.Getenv.
	Getenv(key string) string

	// LookPath must behave like execabs.LookPath.
	LookPath(file string) (string, error)

	// RemoveAll must behave like os.RemoveAll.
	RemoveAll(path string) error

	// StartCmd invokes the cmd.Start.
	StartCmd(cmd *execabs.Cmd) error
}

// DepsStdlib implements [Deps] using the standard library.
type DepsStdlib struct {
	tordatadir.DepsStdlib
}

var _ Deps = DepsStdlib{}

// Getenv implements Deps.
func (DepsStdlib) Getenv(key string) string {
	return os.Getenv(key)
}

// LookPath implements Deps.
func (DepsStdlib) LookPath(file string) (string, error) {
	return execabs.LookPath(file)
}

// RemoveAll implements Deps.
func (DepsStdlib) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// StartCmd implements Deps.
func (DepsStdlib) StartCmd(cmd *exec.Cmd) error {
	return cmd.Start()
}

// Options contains options for [Start].
type Options struct {
	// RemoveDatadir optionally removes the datadir.
	RemoveDatadir bool

	// TorArgs OPTIONALLY allows to append arguments to the command line.
	TorArgs []string

	// TorBinary OPTIONALLY allows to override the binary to execute.
	TorBinary string
}

// Proc is a running tor process instance.
type Proc struct {
	// conn is the control conn.
	conn net.Conn

	// ddstate is the datadir state.
	ddstate *tordatadir.State

	// deps contains the dependencies.
	deps Deps

	// once ensures that Stop is idempotent.
	once *sync.Once

	// options contains the options.
	options *Options

	// p is the process.
	p *os.Process
}

// ControlConn returns the control connection.
func (p *Proc) ControlConn() io.ReadWriteCloser {
	return p.conn
}

// Stop shuts down the process and the control connection.
func (p *Proc) Stop() {
	p.once.Do(func() {
		// make sure we don't leak the control conn.
		_ = p.conn.Close()

		// make sure we don't leak a process ID
		//
		// note: await for exited but eventually kill the process
		// and then make sure we collect the process ID
		exited := make(chan any)
		go func() {
			defer close(exited)
			p.p.Wait()
		}()
		select {
		case <-exited:
			// all is good

		case <-time.After(300 * time.Millisecond):
			// need to use brute force
			p.p.Kill()
			<-exited
		}

		// make sure we close the datadir
		_ = p.ddstate.Close()

		// optionally also zap the whole datadir
		if p.options.RemoveDatadir {
			p.deps.RemoveAll(p.ddstate.DirPath)
		}
	})
}

// Start starts tor using the given data directory.
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
func Start(ctx context.Context, dataDir string, logger model.Logger, options *Options, deps Deps) (*Proc, error) {
	// create the datadir state
	ddstate, err := tordatadir.New(dataDir, deps)
	if err != nil {
		return nil, err
	}

	// fill the default command line arguments
	defaultArgs := []string{
		"--ControlPort", "auto",
		"--ControlPortWriteToFile", ddstate.ControlPortFile,
		"--CookieAuthentication", "1",
		"--CookieAuthFile", ddstate.CookieAuthFile,
		"--DataDirectory", ddstate.DirPath,
		"--DisableNetwork", "1",
		"--SocksPort", "auto",
		"-f", ddstate.TorRcFile,
	}

	// finish assembling the command to exec
	torBinaryPath, err := getTorBinaryPath(deps, options)
	if err != nil {
		_ = ddstate.Close() // make sure we close the datadir
		return nil, err
	}
	cmd := &execabs.Cmd{
		Path:   torBinaryPath,
		Args:   append([]string{torBinaryPath}, append(defaultArgs, options.TorArgs...)...),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	logger.Infof("torexec: + %s", cmd.String())

	// execute the tor binary
	if err := cmd.Start(); err != nil {
		_ = ddstate.Close() // make sure we close the datadir
		return nil, err
	}

	// wait for control connection to be ready
	conn, err := awaitControlConnection(ctx, ddstate)
	if err != nil {
		// make sure we don't leak a process ID
		_ = cmd.Process.Kill()
		defer cmd.Process.Wait()

		// make sure we close the datadir
		_ = ddstate.Close()

		return nil, err
	}

	// we're all good
	p := &Proc{
		conn:    conn,
		ddstate: ddstate,
		deps:    deps,
		once:    &sync.Once{},
		options: options,
		p:       cmd.Process,
	}
	return p, nil
}

// ErrCannotReachControlPort indicates we cannot reach the control port.
var ErrCannotReachControlPort = errors.New("torexec: cannot reach control port")

func awaitControlConnection(ctx context.Context, ddstate *tordatadir.State) (net.Conn, error) {
	dialer := &net.Dialer{}
	for i := 0; i < 10; i++ {
		endpoint, err := readControlPortFile(ddstate)
		if err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		conn, err := dialer.DialContext(ctx, "tcp", endpoint)
		if err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		return conn, nil
	}
	return nil, ErrCannotReachControlPort
}

// errMissingPortPrefix means we're missing the PORT= prefix.
var errMissingPortPrefix = errors.New("torexec: missing the PORT prefix")

func readControlPortFile(ddstate *tordatadir.State) (string, error) {
	data, err := os.ReadFile(ddstate.ControlPortFile)
	if err != nil {
		return "", err
	}
	if !bytes.HasPrefix(data, []byte("PORT=")) {
		return "", errMissingPortPrefix
	}
	endpoint := strings.TrimRight(string(data[5:]), "\r\n")
	return endpoint, nil
}

func getTorBinaryPath(deps Deps, options *Options) (string, error) {
	// ooniTorBinaryEnv is the name of the environment variable
	// we're using to get the path to the tor binary when we are
	// being run by the ooni/probe-desktop application.
	const ooniTorBinaryEnv = "OONI_TOR_BINARY"

	// 1
	if options.TorBinary != "" {
		return deps.LookPath(options.TorBinary)
	}

	// 2
	if binary := deps.Getenv(ooniTorBinaryEnv); binary != "" {
		return binary, nil
	}

	// 3
	return deps.LookPath("tor")
}
