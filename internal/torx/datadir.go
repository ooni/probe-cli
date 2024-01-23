package torx

//
// datadir.go - code to manage tor's data directory.
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

// DataDirDeps contains runtime dependencies for [NewDataDirState].
type DataDirDeps interface {
	// CreateTemp is like os.CreateTemp.
	CreateTemp(dir string, pattern string) (*os.File, error)

	// MkdirAll is like os.MkdirAll.
	MkdirAll(path string, perm fs.FileMode) error

	// Remove is like os.Remove.
	Remove(path string) error

	// RemoveAll is like os.RemoveAll.
	RemoveAll(path string) error
}

// dataDirDepsStdlib implements [DataDirDeps] using the standard library.
type dataDirDepsStdlib struct{}

var _ DataDirDeps = dataDirDepsStdlib{}

// CreateTemp implements DataDirDeps.
func (dataDirDepsStdlib) CreateTemp(dir string, pattern string) (*os.File, error) {
	return os.CreateTemp(dir, pattern)
}

// MkdirAll implements DataDirDeps.
func (dataDirDepsStdlib) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Remove implements DataDirDeps.
func (dataDirDepsStdlib) Remove(path string) error {
	return os.Remove(path)
}

// RemoveAll implements DataDirDeps.
func (dataDirDepsStdlib) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// DataDirOption is an option to configure the [NewDataDirState].
type DataDirOption func(opts *dataDirOptions)

// dataDirOptions contains options for [NewDataDirState].
type dataDirOptions struct {
	deps   DataDirDeps
	remove bool
}

// newDataDirOptions creates [*dataDirOptions] with default settings.
func newDataDirOptions() *dataDirOptions {
	return &dataDirOptions{
		deps:   &dataDirDepsStdlib{},
		remove: false,
	}
}

// DataDirOptionDeps configures alternative [DataDirDeps]. By default we
// use the standard library to implement dependencies.
func DataDirOptionDeps(deps DataDirDeps) DataDirOption {
	return func(opts *dataDirOptions) {
		opts.deps = deps
	}
}

// DataDirOptionRemoveDataDir controls whether the [*DataDirState] Close method
// is going to wipe the whole data directory. The default is to NOT wipe it.
func DataDirOptionRemoveDataDir(value bool) DataDirOption {
	return func(opts *dataDirOptions) {
		opts.remove = value
	}
}

// DataDirState contains the datadir state.
//
// Please, use the [NewDataDirState] factory to construct.
type DataDirState struct {
	// ControlPortFile is the control port file.
	ControlPortFile string

	// CookieAuthFile is the cookie auth file.
	CookieAuthFile string

	// DirPath is the dir path.
	DirPath string

	// TorRcFile is the torrc file.
	TorRcFile string

	// closeOnce provides "once" semantics for Close.
	closeOnce *sync.Once

	// deps contains the dependencies.
	deps DataDirDeps

	// maybeRemoveDataDir is the function that maybe removes
	// the DirPath field when done, depending on the flags that
	// were passed to [NewDataDirState].
	maybeRemoveDataDir func(path string) error
}

// dataDirRemovePolicy selects the proper data dir removal policy
var dataDirRemovePolicy = map[bool]func(deps DataDirDeps) func(path string) error{
	// when the policy is true, we do remove the data directory
	true: func(deps DataDirDeps) func(path string) error {
		return deps.RemoveAll
	},

	// when the policy is false we don't remove the data directory
	false: func(deps DataDirDeps) func(path string) error {
		return func(path string) error {
			return nil
		}
	},
}

// NewDataDirState creates a new [*DataDirState] instance.
func NewDataDirState(dirPath string, options ...DataDirOption) (*DataDirState, error) {
	// honour user provided functional options
	config := newDataDirOptions()
	for _, option := range options {
		option(config)
	}

	// make sure the directory path is absolute
	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, err
	}

	// make sure the directory exists.
	if err := config.deps.MkdirAll(dirPath, 0700); err != nil {
		return nil, err
	}

	// create the controlPortFile.
	controlPortFile, err := config.deps.CreateTemp(dirPath, "control-port-*")
	if err != nil {
		return nil, err
	}
	defer controlPortFile.Close()

	// create the cookieAuthFile.
	cookieAuthFile, err := config.deps.CreateTemp(dirPath, "cookie-auth-*")
	if err != nil {
		return nil, err
	}
	defer cookieAuthFile.Close()

	// create the torRcFile.
	torRcFile, err := config.deps.CreateTemp(dirPath, "torrc-*")
	if err != nil {
		return nil, err
	}
	defer torRcFile.Close()

	// create and return the State struct.
	dd := &DataDirState{
		ControlPortFile:    controlPortFile.Name(),
		CookieAuthFile:     cookieAuthFile.Name(),
		DirPath:            dirPath,
		TorRcFile:          torRcFile.Name(),
		closeOnce:          &sync.Once{},
		deps:               config.deps,
		maybeRemoveDataDir: dataDirRemovePolicy[config.remove](config.deps),
	}
	return dd, nil
}

var _ io.Closer = &DataDirState{}

// Close implements io.Closer.
func (s *DataDirState) Close() error {
	s.closeOnce.Do(func() {
		_ = s.deps.Remove(s.ControlPortFile)
		_ = s.deps.Remove(s.CookieAuthFile)
		_ = s.deps.Remove(s.TorRcFile)
		_ = s.maybeRemoveDataDir(s.DirPath) // conditional on flags
	})
	return nil
}
