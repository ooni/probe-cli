// Package shellxtesting contains mocks for shellx.
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

// RemoveCommonEnvironmentVariables returns the given [execabs.Cmd]
// environment variables minus the ones of the current process.
func RemoveCommonEnvironmentVariables(c *execabs.Cmd) []string {
	const (
		us = 1 << iota
		them
	)
	m := make(map[string]int)
	for _, env := range os.Environ() {
		m[env] |= us
	}
	for _, env := range c.Env {
		m[env] |= them
	}
	out := []string{}
	for key, value := range m {
		if (value & us) == 0 {
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
