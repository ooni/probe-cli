//go:build: cgo

package netxlite

/*
#cgo windows LDFLAGS: -lws2_32

#ifndef _WIN32
#ifdef __linux__
#define __GNU_SOURCE // expose EAI_NODATA on GNU/Linux
#endif
#include <netdb.h> // for getaddrinfo
#else
#include <ws2tcpip.h> // for getaddrinfo
#endif

#include <stdlib.h> // for NULL

// Implementation note: because not all systems define all the
// possible EAI_XXX errors, we need some extra C code to normalize
// system-dependent EAI_XXX errors to the set we care about.

// OONI_EAI_OTHER indicates we know there is a getaddrinfo
// error but we don't know/care about which one.
#define OONI_EAI_OTHER 1

// OONI_EAI_SYSTEM indicates the error is EAI_SYSTEM.
#define OONI_EAI_SYSTEM 2

// OONI_EAI_NONAME indicates the error is EAI_NONAME.
#define OONI_EAI_NONAME 3

// OONI_EAI_NODATA indicates the error is EAI_NODATA.
#define OONI_EAI_NODATA 4

// OONIMapGetaddrinfoError maps a system getaddrinfo error to the
// corresponding portable getaddrinfo error (see above defs).
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

// Implementation note: Windows of course defines getnameinfo with a prototype
// that is not compatible with Unix. So, we need a wrapper function to ensure
// these differences do not end up causing compilation errors.

// OONIAddrinfoToString converts the numeric address inside ainfo into a string
// that is stored inside the host buffer with hostlen bytes.
//
// Because we're using getnameinfo, the https://man.openbsd.org/getnameinfo#CAVEATS
// about using getnameinfo most likely apply here.
static int OONIAddrinfoToString(struct addrinfo *ainfo, char *host, unsigned int hostlen) {
	return getnameinfo(
		ainfo->ai_addr, ainfo->ai_addrlen, // address + length
		host, hostlen,                     // host buffer + length
		NULL, 0,                           // port buffer + length
		NI_NUMERICHOST                     // getnameinfo flags
	);
}
*/
import "C"

import (
	"context"
	"errors"
	"log"
	"net"
	"syscall"
	"unsafe"
)

// getaddrinfoDoLookupHost performs an host lookup with getaddrinfo. Whenever
// possible the returned error is an ErrGetaddrinfo.
func getaddrinfoDoLookupHost(ctx context.Context, domain string) ([]string, error) {
	return getaddrinfoSingleton.Do(ctx, domain)
}

// getaddrinfoSingleton is the getaddrinfo singleton.
var getaddrinfoSingleton = newGetaddrinfoState()

// getaddrinfoSlot is a slot for calling getaddrinfo. The Go standard lib
// limits the maximum number of parallel calls to getaddrinfo. They do that
// to avoid committing too many threads if the system resolver for some
// reason doesn't respond. We need to do the same. Because OONI does not
// need to be as general as the Go stdlib, we'll use a small enough number
// of slots, rather than checking for rlimits, like the stdlib does on
// Unix systems. This struct represents one of these slots.
type getaddrinfoSlot struct{}

// getaddrinfoState is the state associated to getaddrinfo.
type getaddrinfoState struct {
	// sema is the semaphore that only allows a maximum number of
	// getaddrinfo slots to be active at any time.
	sema chan *getaddrinfoSlot
}

// getaddrinfoNumSlots is the maximum number of parallel calls
// to getaddrinfo we may have at any given time.
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

// grabSlot grabs a slot for calling getaddrinfo. This function may block until
// a slot becomes available (or until the context is done).
func (state *getaddrinfoState) grabSlot(ctx context.Context) error {
	// Implementation note: the channel has getaddrinfoNumSlots capacity, hence
	// the first getaddrinfoNumSlots channel writes will succeed and all the
	// subsequent onces will block. To unblock a pending request, we release a
	// slot by reading from the channel.
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
// slot and you're defer-releasing it when you're done.
//
// This function is adapted from cgoLookupIPCNAME
// https://github.com/golang/go/blob/go1.17.6/src/net/cgo_unix.go#L145
//
// SPDX-License-Identifier: BSD-3-Clause.
func (state *getaddrinfoState) do(domain string) ([]string, error) {
	var hints C.struct_addrinfo     // zero-initialized by Go
	hints.ai_flags = C.AI_CANONNAME // TODO(bassosimone): we can do better here
	hints.ai_socktype = C.SOCK_STREAM
	hints.ai_family = C.AF_UNSPEC
	h := make([]byte, len(domain)+1)
	copy(h, domain)
	var res *C.struct_addrinfo
	// From https://pkg.go.dev/cmd/cgo:
	//
	// "Any C function (even void functions) may be called in a multiple
	// assignment context to retrieve both the return value (if any) and the
	// C errno variable as an error"
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
	// TODO(bassosimone): as long as we're testing this new functionality
	// we will keep a bit more logging to help in diagnosing errors. (Note
	// that here err _may_ be nil if we only have a getaddrinfo failure and
	// there was no actual syscall error.)
	log.Printf("getaddrinfo: code=%d err=%+v", code, err)
	switch C.OONIMapGetaddrinfoError(code) {
	case C.OONI_EAI_SYSTEM:
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
		return nil, err
	case C.OONI_EAI_NONAME:
		err = errors.New(DNSNoSuchHostSuffix) // so it becomes ErrDNSNXDOMAIN
		return nil, newErrGetaddrinfo(int64(code), err)
	case C.OONI_EAI_NODATA:
		// We have seen that on Android systems NXDOMAIN maps to
		// EAI_NODATA and it's unclear whether this is the case for
		// other systems as well. As far as we know, this does not
		// happen for GNU/Linux, Windows, and macOS but it may be
		// the case for some other systems (maybe BSD?).
		//
		// So the design choice here is to map to NXDOMAIN and
		// also to allow extracting the original Getaddrinfo code,
		// to provide low-level information of the real error.
		//
		// See https://github.com/ooni/probe/issues/2029 for the
		// investigation regarding Android systems.
		//
		// TODO(bassosimone): before releasing this code in
		// prod, we should decide whether this is OK.
		err = errors.New(DNSNoSuchHostSuffix) // so it becomes FailureDNSNXDOMAINError
		return nil, newErrGetaddrinfo(int64(code), err)
	default:
		err = errors.New(DNSServerMisbehavingSuffix) // so it becomes FailureDNSServerMisbehaving
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
		// Implementation note: the original Go stdlib code used another
		// strategy for mapping to net.IPAddr that required copying
		// more code from the stdlib to ensure IPv6 scopes are properly
		// dealt with. So, here we're actually taking a shortcut and
		// we will just call getnameinfo to reverse map to string.
		addr, err := state.addrintoToString(r)
		if err != nil {
			continue
		}
		// According to getnameinfo's manual CAVEATS section for OpenBSD
		// (https://man.openbsd.org/getnameinfo#CAVEATS), it's possible to
		// trick getnameinfo into returning non-IP address values by
		// registering records such as
		//
		//     1.0.0.127.in-addr.arpa. IN PTR  10.1.1.1
		//
		// The manual page then recommends using NI_NAMEDREQ for a first
		// getnameinfo call, then followed by a NI_NUMERICHOST call.
		//
		// Here we're using a different, but equivalent, strategy where
		// we don't include addr if it's not a valid IP addr.
		if net.ParseIP(addr) == nil {
			continue
		}
		addrs = append(addrs, addr)
	}
	if len(addrs) < 1 {
		// TODO(bassosimone): as long as we're testing this new functionality
		// we will keep a bit more logging to help in diagnosing errors.
		log.Printf("getaddrinfo: no address after getnameinfo loop")
		return nil, errors.New(DNSNoAnswerSuffix)
	}
	return addrs, nil
}

// errGetnameinfoFailed means that getnameinfo unexpectedly failed.
var errGetnameinfoFailed = errors.New("getnameinfo failed")

// addrinfoToString obtains a string represenation of r. Because we're using
// getnameinfo, see https://man.openbsd.org/getnameinfo#CAVEATS.
func (state *getaddrinfoState) addrintoToString(r *C.struct_addrinfo) (string, error) {
	// Implementation note: NI_MAXHOST is documented as the maximum size of
	// the buffer, and all examples allocate a buffer of that size, including
	// e.g., https://man.openbsd.org/getnameinfo#EXAMPLES.
	const hostbufsiz = C.NI_MAXHOST
	hostbuf := make([]byte, hostbufsiz)
	res := C.OONIAddrinfoToString(
		r,                                      // addrinfo
		(*C.char)(unsafe.Pointer(&hostbuf[0])), // buffer
		hostbufsiz,                             // buffer size
	)
	if res != 0 {
		// TODO(bassosimone): as long as we're testing this new functionality
		// we will keep a bit more logging to help in diagnosing errors.
		log.Printf("getnameinfo: code=%d", res)
		return "", errGetnameinfoFailed
	}
	return C.GoString((*C.char)(unsafe.Pointer(&hostbuf[0]))), nil
}
