// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build cgo && linux && !android

package netxlite

/*
#include <netdb.h>
*/
import "C"

import (
	"errors"
	"syscall"
)

// NOTE(rsc): In theory there are approximately balanced
// arguments for and against including AI_ADDRCONFIG
// in the flags (it includes IPv4 results only on IPv4 systems,
// and similarly for IPv6), but in practice setting it causes
// getaddrinfo to return the wrong canonical name on Linux.
// So definitely leave it out.
const getaddrinfoAIFlags = C.AI_CANONNAME | C.AI_V4MAPPED | C.AI_ALL

// toError is the function that converts the return value from
// the getaddrinfo function into a proper Go error.
//
// This function is adapted from cgoLookupIPCNAME
// https://github.com/golang/go/blob/go1.17.6/src/net/cgo_unix.go#L145
//
// SPDX-License-Identifier: BSD-3-Clause.
func (state *getaddrinfoState) toError(code C.int, err error) ([]string, string, error) {
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
		return nil, "", newErrGetaddrinfo(int64(code), err)
	case C.EAI_NONAME:
		err = errors.New(DNSNoSuchHostSuffix) // so it becomes FailureDNSNXDOMAIN
		return nil, "", newErrGetaddrinfo(int64(code), err)
	default:
		err = errors.New(DNSServerMisbehavingSuffix) // so it becomes FailureDNSServerMisbehaving
		return nil, "", newErrGetaddrinfo(int64(code), err)
	}
}
