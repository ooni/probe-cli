// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build cgo && android

package netxlite

//#include <netdb.h>
import "C"

const getaddrinfoAIFlags = C.AI_CANONNAME

// toError is the function that converts the return value from
// the getaddrinfo function into a proper Go error.
//
// This function is adapted from cgoLookupIPCNAME
// https://github.com/golang/go/blob/go1.17.6/src/net/cgo_unix.go#L145
//
// SPDX-License-Identifier: BSD-3-Clause.
func (state *getaddrinfoState) toError(code C.int, err error) ([]string, error) {
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
		return nil, newErrGetaddrinfo(int64(code), err)
	case C.EAI_NONAME, C.EAI_NODATA:
		// We have seen that on Android systems NXDOMAIN maps to
		// EAI_NODATA and it's unclear whether this is the case for
		// other systems as well. As far as we know, this does not
		// happen for GNU/Linux, Windows, and macOS but it may be
		// the case for some other systems (maybe BSD?).
		//
		// So the design choice here is to map to NXDOMAIN for
		// robustness but _also_ to record the original getaddrinfo
		// code, so one can see it into the results.
		//
		// See https://github.com/ooni/probe/issues/2029 for the
		// investigation on Android's getaddrinfo.
		err = errors.New(DNSNoSuchHostSuffix) // so it becomes ErrDNSNXDOMAIN
		return nil, newErrGetaddrinfo(int64(code), err)
	default:
		err = errors.New(DNSServerMisbehavingSuffix) // so it becomes FailureDNSServerMisbehaving
		return nil, newErrGetaddrinfo(int64(code), err)
	}
}
