// Package shellxtesting supports shellx testing.
package shellxtesting

import (
	"os"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"golang.org/x/sys/execabs"
)

// Library implements shellx.Dependencies.
type Library struct {
	MockCmdOutput func(c *execabs.Cmd) ([]byte, error)

	MockCmdRun func(c *execabs.Cmd) error

	MockLookPath func(file string) (string, error)
}

var _ shellx.Dependencies = &Library{}

// CmdOutput implements shellx.Dependencies
func (lib *Library) CmdOutput(c *execabs.Cmd) ([]byte, error) {
	return lib.MockCmdOutput(c)
}

// CmdRun implements shellx.Dependencies
func (lib *Library) CmdRun(c *execabs.Cmd) error {
	return lib.MockCmdRun(c)
}

// LookPath implements shellx.Dependencies
func (lib *Library) LookPath(file string) (string, error) {
	return lib.MockLookPath(file)
}

// MustArgv returns the [execabs.Cmd]'s Argv or panics.
func MustArgv(c *execabs.Cmd) []string {
	runtimex.Assert(len(c.Args) >= 1, "too few arguments")
	out := []string{c.Path}
	out = append(out, c.Args[1:]...)
	return out
}

// CmdEnvironMinusOsEnviron removes the environment variables in
// [os.Environ] from the ones inside the given command. Note that
// the variables in os.Environ and in the command are like name=value,
// therefore, if you have HOME=/home/sbs in os.Environ and have
// HOME=/tmp in the command, you'll get HOME=/tmp in output.
func CmdEnvironMinusOsEnviron(c *execabs.Cmd) []string {
	const (
		inCmd = 1 << iota
		inEnviron
	)
	m := make(map[string]int)
	for _, env := range os.Environ() {
		m[env] |= inEnviron
	}
	for _, env := range c.Env {
		m[env] |= inCmd
	}
	out := []string{}
	for key, value := range m {
		if value == inCmd {
			out = append(out, key)
		}
	}
	return out
}

// WithCustomLibrary executes the given function with a custom shellx.Library.
func WithCustomLibrary(library shellx.Dependencies, fn func()) {
	prev := shellx.Library
	defer func() {
		shellx.Library = prev
	}()
	shellx.Library = library
	fn()
}
