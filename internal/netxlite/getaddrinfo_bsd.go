// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build cgo && (darwin || dragonfly || freebsd || openbsd)

package netxlite

/*
#include <netdb.h>
*/
import "C"
import (
	"syscall"
)

const getaddrinfoAIFlags = (C.AI_CANONNAME | C.AI_V4MAPPED | C.AI_ALL) & C.AI_MASK

// Making constants available to Go code so we can run tests
const (
	aiCanonname = C.AI_CANONNAME
	aiV4Mapped  = C.AI_V4MAPPED
	aiAll       = C.AI_ALL
	aiMask      = C.AI_MASK
	eaiSystem   = C.EAI_SYSTEM
	eaiNoName   = C.EAI_NONAME
	eaiBadFlags = C.EAI_BADFLAGS
)

// toError is the function that converts the return value from
// the getaddrinfo function into a proper Go error.
//
// This function is adapted from cgoLookupIPCNAME
// https://github.com/golang/go/blob/go1.17.6/src/net/cgo_unix.go#L145
//
// SPDX-License-Identifier: BSD-3-Clause.
func (state *getaddrinfoState) toError(code int64, err error, goos string) error {
	switch code {
	case C.EAI_SYSTEM:
		if err == nil {
			// err should not be nil, but sometimes getaddrinfo returns
			// code == C.EAI_SYSTEM with err == nil on Linux.
			// The report claims that it happens when we have too many
			// open files, so use syscall.EMFILE (too many open files in system).
			// Most system calls would return ENFILE (too many open files),
			// so at the least EMFILE should be easy to recognize if this
			// comes up again. golang.org/issue/6232.
			err = syscall.EMFILE
		}
		return newErrGetaddrinfo(code, err)
	case C.EAI_NONAME:
		err = ErrOODNSNoSuchHost // so it becomes FailureDNSNXDOMAIN
		return newErrGetaddrinfo(code, err)
	default:
		err = ErrOODNSMisbehaving // so it becomes FailureDNSServerMisbehaving
		return newErrGetaddrinfo(code, err)
	}
}
