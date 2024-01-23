// Package tordatadir contains code to manage tor data directory.
package tordatadir

//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Deps contains runtime dependencies for [New].
type Deps interface {
	// CreateTemp is like os.CreateTemp.
	CreateTemp(dir string, pattern string) (*os.File, error)

	// MkdirAll is like os.MkdirAll.
	MkdirAll(path string, perm fs.FileMode) error

	// Remove is like os.Remove.
	Remove(name string) error
}

// DepsStdlib implements [Deps] using the standard library.
type DepsStdlib struct{}

var _ Deps = DepsStdlib{}

// CreateTemp implements Deps.
func (DepsStdlib) CreateTemp(dir string, pattern string) (*os.File, error) {
	return os.CreateTemp(dir, pattern)
}

// MkdirAll implements Deps.
func (DepsStdlib) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Remove implements Deps.
func (DepsStdlib) Remove(name string) error {
	return os.Remove(name)
}

// State contains the datadir state.
//
// Please, use the [New] factory to construct.
type State struct {
	// closeOnce provides "once" semantics for Close.
	closeOnce *sync.Once

	// ControlPortFile is the control port file.
	ControlPortFile string

	// CookieAuthFile is the cookie auth file.
	CookieAuthFile string

	// deps contains the dependencies.
	deps Deps

	// DirPath is the dir path.
	DirPath string

	// TorRcFile is the torrc file.
	TorRcFile string
}

// New creates a new [*State] instance.
func New(dirPath string, deps Deps) (*State, error) {
	// make sure the directory path is absolute
	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, err
	}

	// make sure the directory exists.
	if err := deps.MkdirAll(dirPath, 0700); err != nil {
		return nil, err
	}

	// create the controlPortFile.
	controlPortFile, err := deps.CreateTemp(dirPath, "control-port-*")
	if err != nil {
		return nil, err
	}
	defer controlPortFile.Close()

	// create the cookieAuthFile.
	cookieAuthFile, err := deps.CreateTemp(dirPath, "cookie-auth-*")
	if err != nil {
		return nil, err
	}
	defer cookieAuthFile.Close()

	// create the torRcFile.
	torRcFile, err := deps.CreateTemp(dirPath, "torrc-*")
	if err != nil {
		return nil, err
	}
	defer torRcFile.Close()

	// create and return the State struct.
	dd := &State{
		closeOnce:       &sync.Once{},
		ControlPortFile: controlPortFile.Name(),
		CookieAuthFile:  cookieAuthFile.Name(),
		deps:            deps,
		DirPath:         dirPath,
		TorRcFile:       torRcFile.Name(),
	}
	return dd, nil
}

var _ io.Closer = &State{}

// Close implements io.Closer.
func (s *State) Close() error {
	s.closeOnce.Do(func() {
		_ = s.deps.Remove(s.ControlPortFile)
		_ = s.deps.Remove(s.CookieAuthFile)
		_ = s.deps.Remove(s.TorRcFile)
	})
	return nil
}
