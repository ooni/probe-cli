//go:build: cgo

package netxlite

/*
#cgo windows LDFLAGS: -lws2_32

#ifndef _WIN32
#define __GNU_SOURCE // for EAI_NODATA on GNU/Linux
#include <netdb.h>
#else
#include <ws2tcpip.h>
#endif

#include <stdlib.h>

#define OONI_EAI_OTHER  1 // means: any other error
#define OONI_EAI_SYSTEM 2 // means: EAI_SYSTEM
#define OONI_EAI_NONAME 3 // means: EAI_NONAME
#define OONI_EAI_NODATA 4 // means: EAI_NODATA

// OONIMapGetaddrinfoError maps a getaddrinfo error in the
// system domain to a normalized error (see above defs).
static int OONIMapGetaddrinfoError(int rv) {
	switch (rv) {
#ifdef EAI_SYSTEM // not available on, e.g, Windows
	case EAI_SYSTEM:
		return OONI_EAI_SYSTEM;
#endif
	case EAI_NONAME:
		return OONI_EAI_NONAME;
#ifdef EAI_NODATA // not available on, e.g., Windows
	case EAI_NODATA:
		return OONI_EAI_NODATA;
#endif
	default:
		return OONI_EAI_OTHER;
	}
}

static int OONIGetNameInfo(struct addrinfo *ainfo, char *host, unsigned int hostlen) {
	return getnameinfo(
		ainfo->ai_addr, ainfo->ai_addrlen, host, hostlen, NULL, 0, NI_NUMERICHOST);
}
*/
import "C"

import (
	"context"
	"errors"
	"syscall"
	"unsafe"
)

// getaddrinfoAvailable returns whether getaddrinfo is available.
func getaddrinfoAvailable() bool {
	return true
}

// getaddrinfoDoLookupHost performs an host lookup with getaddrinfo.
func getaddrinfoDoLookupHost(ctx context.Context, domain string) ([]string, error) {
	return getaddrinfoSingleton.Do(ctx, domain)
}

// getaddrinfoSingleton is the getaddrinfo singleton.
var getaddrinfoSingleton = newGetaddrinfoState()

// getaddrinfoSlot is a slot for calling getaddrinfo.
type getaddrinfoSlot struct{}

// getaddrinfoState is the state associated to getaddrinfo.
type getaddrinfoState struct {
	sema chan *getaddrinfoSlot
}

// getaddrinfoNumSlots is the maximum number of parallel
// calls to getaddrinfo we may have at any given time.
//
// TODO(bassosimone): better document strategy.
const getaddrinfoNumSlots = 8

// newGetaddrinfoState creates the getaddrinfo state.
func newGetaddrinfoState() *getaddrinfoState {
	return &getaddrinfoState{
		sema: make(chan *getaddrinfoSlot, getaddrinfoNumSlots),
	}
}

// Do invokes getaddrinfo and returns the results.
func (state *getaddrinfoState) Do(ctx context.Context, domain string) ([]string, error) {
	if err := state.grabSlot(ctx); err != nil {
		return nil, err
	}
	defer state.releaseSlot()
	return state.do(domain)
}

// grabSlot grabs a slot for calling getaddrinfo.
func (state *getaddrinfoState) grabSlot(ctx context.Context) error {
	select {
	case state.sema <- &getaddrinfoSlot{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// releaseSlot releases a previously acquired slot.
func (state *getaddrinfoState) releaseSlot() {
	<-state.sema
}

// do calls getaddrinfo. We assume that you've already grabbed a
// slot and you're releasing it when this function returns.
//
// This function is adapted from cgoLookupIPCNAME
// https://github.com/golang/go/blob/go1.17.6/src/net/cgo_unix.go#L145
//
// SPDX-License-Identifier: BSD-3-Clause.
func (state *getaddrinfoState) do(domain string) ([]string, error) {
	var hints C.struct_addrinfo
	hints.ai_flags = C.AI_CANONNAME // TODO(bassosimone): we can do better here
	hints.ai_socktype = C.SOCK_STREAM
	hints.ai_family = C.AF_UNSPEC
	h := make([]byte, len(domain)+1)
	copy(h, domain)
	var res *C.struct_addrinfo
	code, err := C.getaddrinfo((*C.char)(unsafe.Pointer(&h[0])), nil, &hints, &res)
	if code != 0 {
		return state.toError(code, err)
	}
	defer C.freeaddrinfo(res)
	return state.toAddressList(res)
}

// toError is the function that converts the return value from
// the getaddrinfo function into a proper Go error.
//
// This function is adapted from cgoLookupIPCNAME
// https://github.com/golang/go/blob/go1.17.6/src/net/cgo_unix.go#L145
//
// SPDX-License-Identifier: BSD-3-Clause.
func (state *getaddrinfoState) toError(code C.int, err error) ([]string, error) {
	// Implementation note: we need to map the error returned by
	// getaddrinfo to the set of errors we understand. On GNU/Linux
	// it seems there's no EAI_NODATA unless __GNU_SOURCE and it
	// also seems defining it does not change what the CGO will
	// understand. So we use a C function for performing mapping.
	switch C.OONIMapGetaddrinfoError(code) {
	case C.OONI_EAI_SYSTEM:
		if err == nil {
			// err should not be nil, but sometimes getaddrinfo returns
			// gerrno == C.EAI_SYSTEM with err == nil on Linux.
			// The report claims that it happens when we have too many
			// open files, so use syscall.EMFILE (too many open files in system).
			// Most system calls would return ENFILE (too many open files),
			// so at the least EMFILE should be easy to recognize if this
			// comes up again. golang.org/issue/6232.
			err = syscall.EMFILE
		}
		return nil, err
	case C.OONI_EAI_NONAME:
		err = errors.New(DNSNoSuchHostSuffix)
		return nil, newErrGetaddrinfo(int64(code), err)
	case C.OONI_EAI_NODATA:
		// We have seen that on Android systems NXDOMAIN maps to
		// EAI_NODATA and it's unclear whether this is the case for
		// other systems as well. As far as we know, this does not
		// happen for GNU/Linux, Windows, and macOS but it may be
		// the case for some other systems (maybe BSD?).
		//
		// So the design choice here is to map to NXDOMAIN and
		// also to allow extracting the original Getaddrinfo code.
		//
		// See https://github.com/ooni/probe/issues/2029 for the
		// investigation regarding Android systems.
		err = errors.New(DNSNoSuchHostSuffix)
		return nil, newErrGetaddrinfo(int64(code), err)
	default:
		err = errors.New(DNSServerMisbehavingSuffix)
		return nil, newErrGetaddrinfo(int64(code), err)
	}
}

// toAddressList is the function that converts the return value from
// the getaddrinfo function into a list of strings.
//
// This function is adapted from cgoLookupIPCNAME
// https://github.com/golang/go/blob/go1.17.6/src/net/cgo_unix.go#L145
//
// SPDX-License-Identifier: BSD-3-Clause.
func (state *getaddrinfoState) toAddressList(res *C.struct_addrinfo) ([]string, error) {
	var addrs []string
	for r := res; r != nil; r = r.ai_next {
		// We only asked for SOCK_STREAM, but check anyhow.
		if r.ai_socktype != C.SOCK_STREAM {
			continue
		}
		// Implementation note: the original libc code used another
		// strategy for mapping to net.IPAddr, but that required copying
		// more code from the stdlib. So, here we're actually taking a
		// shortcut and we will just call getnameinfo.
		addr, err := state.getnameinfo(r)
		if err != nil {
			continue
		}
		addrs = append(addrs, addr)
	}
	if len(addrs) < 1 {
		return nil, errors.New(DNSNoAnswerSuffix)
	}
	return addrs, nil
}

// getnameinfo calls the getnameinfo C function.
func (state *getaddrinfoState) getnameinfo(r *C.struct_addrinfo) (string, error) {
	hostbuf := make([]byte, C.NI_MAXHOST)
	res := C.OONIGetNameInfo(r, (*C.char)(unsafe.Pointer(&hostbuf[0])), C.NI_NUMERICHOST)
	if res != 0 {
		return "", errors.New("getnameinfo failed")
	}
	return C.GoString((*C.char)(unsafe.Pointer(&hostbuf[0]))), nil
}
