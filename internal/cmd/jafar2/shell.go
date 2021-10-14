package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/shlex"
	"golang.org/x/sys/execabs"
)

// Shell is a generic shell.
type Shell interface {
	// DefaultGatewayDevice returns the default gateway device.
	DefaultGatewayDevice() (string, error)

	// MkdirAll creates directories recursively.
	MkdirAll(path string, perm os.FileMode) error

	// RemoveAll is equivalent to rm -rf
	RemoveAll(path string) error

	// Run runs the given command line.
	Run(cmdline string) error

	// Runv runs the given argv.
	Runv(argv []string) error

	// WriteFile writes into a file.
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// NewShell creates the new shell compatible with the Environment.
func NewShell(env *Environment) Shell {
	if env.DryRun {
		return &NopShell{}
	}
	return &LinuxShell{}
}

// log logs that we're about to run a command.
func log(cmdline string) {
	fmt.Fprintf(os.Stderr, "+ %s\n", strings.TrimRight(cmdline, " \r\n"))
}

// logf formats and logs a command without executing it.
func logf(format string, v ...interface{}) {
	log(fmt.Sprintf(format, v...))
}

// ShellRunf formats a command line and runs it.
func ShellRunf(sh Shell, format string, v ...interface{}) error {
	return sh.Run(fmt.Sprintf(format, v...))
}

// LinuxShell is the Shell for Linux.
type LinuxShell struct{}

var _ Shell = &LinuxShell{}

// DefaultGatewayDevice returns the default gateway device.
func (sh *LinuxShell) DefaultGatewayDevice() (string, error) {
	out, err := sh.output("ip route show default")
	if err != nil {
		return "", err
	}
	log("awk '{print $5}'")
	lines := bytes.Split(out, []byte("\n"))
	if len(lines) != 2 {
		return "", errors.New("unexpected number of lines")
	}
	if len(lines[1]) != 0 {
		return "", errors.New("unexpected number of lines")
	}
	line := lines[0]
	tokens := bytes.Split(line, []byte(" "))
	if len(tokens) != 10 {
		return "", errors.New("unexpected number of tokens")
	}
	if len(tokens[9]) != 0 {
		return "", errors.New("unexpected number of tokens")
	}
	return string(tokens[4]), nil
}

// MkdirAll creates directories recursively.
func (sh *LinuxShell) MkdirAll(path string, perm os.FileMode) error {
	logf("mkdir -pm %#o %s", perm, path)
	return os.MkdirAll(path, perm)
}

// output gets the output of a command.
func (sh *LinuxShell) output(cmdline string) ([]byte, error) {
	arguments, err := shlex.Split(cmdline)
	if err != nil {
		return nil, err
	}
	cmd, err := sh.preparev(arguments)
	if err != nil {
		return nil, err
	}
	cmd.Stdin, cmd.Stderr = os.Stdin, os.Stderr
	return cmd.Output()
}

// RemoveAll is equivalent to rm -rf
func (sh *LinuxShell) RemoveAll(path string) error {
	logf("rm -rf %s", path)
	return os.RemoveAll(path)
}

// Run runs the given command line.
func (sh *LinuxShell) Run(cmdline string) error {
	arguments, err := shlex.Split(cmdline)
	if err != nil {
		return err
	}
	return sh.Runv(arguments)
}

// Runv runs the given argv.
func (sh *LinuxShell) Runv(argv []string) error {
	cmd, err := sh.preparev(argv)
	if err != nil {
		return err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()

}

// WriteFile writes into a file.
func (sh *LinuxShell) WriteFile(name string, data []byte, perm os.FileMode) error {
	logf("echo $data > %s", name)
	logf("chmod %#o %s", perm, name)
	return os.WriteFile(name, data, perm)
}

// preparev prepares for running given an argv
func (sh *LinuxShell) preparev(argv []string) (*execabs.Cmd, error) {
	if len(argv) < 1 {
		return nil, errors.New("no command specified")
	}
	cmd := execabs.Command(argv[0], argv[1:]...)
	logf("%s\n", cmd.String())
	return cmd, nil
}

// NopShell is a Shell that does nothing.
type NopShell struct{}

var _ Shell = &NopShell{}

// DefaultGatewayDevice returns the default gateway device.
func (sh *NopShell) DefaultGatewayDevice() (string, error) {
	logf("ip route show default")
	log("awk '{print $5}'")
	return "eth0", nil
}

// MkdirAll creates directories recursively.
func (sh *NopShell) MkdirAll(path string, perm os.FileMode) error {
	logf("mkdir -pm %#o %s", perm, path)
	return nil
}

// RemoveAll is equivalent to rm -rf
func (sh *NopShell) RemoveAll(path string) error {
	logf("rm -rf %s", path)
	return nil
}

// Run runs the given command line.
func (sh *NopShell) Run(cmdline string) error {
	arguments, err := shlex.Split(cmdline)
	if err != nil {
		return err
	}
	return sh.Runv(arguments)
}

// Runv runs the given argv.
func (sh *NopShell) Runv(argv []string) error {
	cmd, err := sh.preparev(argv)
	if err != nil {
		return err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return nil

}

// WriteFile writes into a file.
func (sh *NopShell) WriteFile(name string, data []byte, perm os.FileMode) error {
	logf("echo $data > %s", name)
	logf("chmod %#o %s", perm, name)
	return nil
}

// preparev prepares for running given an argv
func (sh *NopShell) preparev(argv []string) (*execabs.Cmd, error) {
	if len(argv) < 1 {
		return nil, errors.New("no command specified")
	}
	cmd := execabs.Command(argv[0], argv[1:]...)
	logf("%s\n", cmd.String())
	return cmd, nil
}
