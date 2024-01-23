package torx

//
// library.go - generic code to use tor as a library.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/apex/log"
)

var LibraryExecUnsafe = libraryExec

// LibrarySingletonDirsProvider is [LibrarySingletonCreate] view of a directory provider.
type LibrarySingletonDirsProvider interface {
	TunnelDir() string
}

// variables to manage the library singleton process.
var (
	librarySingletonConn    *ControlConn
	librarySingletonMu      = &sync.Mutex{}
	librarySingletonProcess Process
)

// ErrLibrarySingletonAlreadyExists means that a library singleton has
// already been created. There can only be a single instance of the library
// process because libtor.a aborts when tor_run_main is invoked multiple
// times in a row, as documented by [ooni/probe#2046].
//
// [ooni/probe#2406]: https://github.com/ooni/probe/issues/2406.
var ErrLibrarySingletonAlreadyExists = errors.New(
	"torx: library singleton already exists",
)

// LibrarySingletonCreate creates the libtor.a library singleton. There can only
// be a single instance of the library process because libtor.a aborts when tor_run_main
// is invoked multiple times in a row, as documented by [ooni/probe#2046].
//
// [ooni/probe#2406]: https://github.com/ooni/probe/issues/2406.
func LibrarySingletonCreate(ctx context.Context, ldp LibrarySingletonDirsProvider) error {
	// mutual exclusion
	defer librarySingletonMu.Unlock()
	librarySingletonMu.Lock()

	// stop if it already exists
	if librarySingletonProcess != nil {
		return ErrLibrarySingletonAlreadyExists
	}

	// create singleton dir
	singletondir := filepath.Join(ldp.TunnelDir(), "tor-singleton")

	// create the data dir
	datadir, err := NewDataDirState(singletondir)
	if err != nil {
		return err
	}

	// create the singleton
	proc, err := libraryExec(datadir, log.Log)
	if err != nil {
		return err
	}

	// obtain a control connection
	conn, err := proc.DialControl(ctx)
	if err != nil {
		return err
	}

	// authenticate using the safe cookie mechanism
	if err := AuthenticateFlowWithSafeCookie(ctx, conn); err != nil {
		return err
	}

	// save the singleton
	librarySingletonConn = conn
	librarySingletonProcess = proc
	return nil
}

// ErrNoLibrarySingleton indicates that you have not created a library singleton yet.
var ErrNoLibrarySingleton = errors.New("torx: no library singleton")

// LibrarySingletonBootstrap performs a bootstrap using the library singleton.
func LibrarySingletonBootstrap(ctx context.Context) ([]string, error) {
	defer librarySingletonMu.Unlock()
	librarySingletonMu.Lock()

	if librarySingletonConn == nil {
		return nil, ErrNoLibrarySingleton
	}

	return Bootstrap(ctx, librarySingletonConn)
}

// LibrarySingletonSetConf sends SETCONF to the library singleton.
func LibrarySingletonSetConf(ctx context.Context, values ...*KeyValuePair) error {
	defer librarySingletonMu.Unlock()
	librarySingletonMu.Lock()

	if librarySingletonConn == nil {
		return ErrNoLibrarySingleton
	}

	return SetConf(ctx, librarySingletonConn, values...)
}

// LibrarySingletonGetInfo sends GETINFO to the library singleton.
func LibrarySingletonGetInfo(ctx context.Context, key string) ([]*KeyValuePair, error) {
	defer librarySingletonMu.Unlock()
	librarySingletonMu.Lock()

	if librarySingletonConn == nil {
		return nil, ErrNoLibrarySingleton
	}

	return GetInfo(ctx, librarySingletonConn, key)
}

// LibrarySingletonProtocolInfo sends PROTOCOLINFO to the library singleton.
func LibrarySingletonProtocolInfo(ctx context.Context) (*ProtocolInfoResponse, error) {
	defer librarySingletonMu.Unlock()
	librarySingletonMu.Lock()

	if librarySingletonConn == nil {
		return nil, ErrNoLibrarySingleton
	}

	return ProtocolInfo(ctx, librarySingletonConn)
}

// TODO(bassosimone): I am starting to wonder if this approach
// of creating specific wrappers is better than just exposing the
// underlying conn used by libtor.a in terms of code reuse.
//
// Yeah, like this it's not composable at all...
//
// In any case, now my objective is to understand how I can
// configure tor such that we can build circuits given existing
// circuits and I suppose there is a way to do that.

// LibrarySingletonSignal sends SIGNAL NEWNYM to the library singleton.
func LibrarySingletonSignalNewNym(ctx context.Context) error {
	defer librarySingletonMu.Unlock()
	librarySingletonMu.Lock()

	if librarySingletonConn == nil {
		return ErrNoLibrarySingleton
	}

	_, err := librarySingletonConn.SendRecv(ctx, "SIGNAL NEWNYM")
	return err
}

type CircuitInfo struct {
	ID     int
	Status string
}

// LibrarySingletonGetInfoCircuitStatus ...
func LibrarySingletonGetInfoCircuitStatus(ctx context.Context) ([]*CircuitInfo, error) {
	pairs, err := LibrarySingletonGetInfo(ctx, "circuit-status")
	if err != nil {
		return nil, err
	}
	result := []*CircuitInfo{}

	for _, pair := range pairs {
		if pair.Key != "circuit-status" {
			continue
		}
		if pair.Value.IsNone() {
			continue
		}
		value := pair.Value.Unwrap()
		for _, line := range strings.Split(value, "\n") {
			tokens := strings.Split(line, " ")
			if len(tokens) < 2 {
				continue
			}
			ID, err := strconv.Atoi(tokens[0])
			if err != nil {
				continue
			}
			result = append(result, &CircuitInfo{
				ID:     ID,
				Status: tokens[1],
			})
		}
	}

	return result, nil
}

// LibrarySingletonCloseCircuits...
func LibrarySingletonCloseCircuits(ctx context.Context, IDs ...int) error {
	defer librarySingletonMu.Unlock()
	librarySingletonMu.Lock()

	if librarySingletonConn == nil {
		return ErrNoLibrarySingleton
	}

	for _, ID := range IDs {
		if _, err := librarySingletonConn.SendRecv(ctx, "CLOSECIRCUIT %d", ID); err != nil {
			return err
		}

	}

	return nil
}
