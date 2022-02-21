//go:build: cgo

package netxlite

/*
// On Unix systems, getaddrinfo is part of libc. On Windows,
// instead, we need to explicitly link with winsock2.
#cgo windows LDFLAGS: -lws2_32

#ifndef _WIN32
#include <netdb.h> // for getaddrinfo
#else
#include <ws2tcpip.h> // for getaddrinfo
#endif
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

func getaddrinfoDoLookupHost(ctx context.Context, domain string) ([]string, error) {
	return getaddrinfoSingleton.Do(ctx, domain)
}

// getaddrinfoSingleton is the getaddrinfo singleton.
var getaddrinfoSingleton = newGetaddrinfoState()

// getaddrinfoSlot is a slot for calling getaddrinfo. The Go standard lib
// limits the maximum number of parallel calls to getaddrinfo. They do that
// to avoid using too many threads if the system resolver for some
// reason doesn't respond. We need to do the same. Because OONI does not
// need to be as general as the Go stdlib, we'll use a small-enough number
// of slots, rather than checking for rlimits, like the stdlib does,
// e.g., on Unix. This struct represents one of these slots.
type getaddrinfoSlot struct{}

// getaddrinfoState is the state associated to getaddrinfo.
type getaddrinfoState struct {
	// sema is the semaphore that only allows a maximum number of
	// getaddrinfo slots to be active at any given time.
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
	var hints C.struct_addrinfo // zero-initialized by Go
	hints.ai_flags = getaddrinfoAIFlags
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
		// TODO(bassosimone): as long as we're testing this new functionality
		// we will keep a bit more logging to help in diagnosing errors. (Note
		// that here err _may_ be nil if we only have a getaddrinfo failure and
		// there was no actual syscall error, hence we use "%+v".)
		log.Printf("getaddrinfo: code=%d err=%+v", code, err)
		return state.toError(code, err)
	}
	defer C.freeaddrinfo(res)
	return state.toAddressList(res)
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
		addr, err := state.addrinfoToString(r)
		if err != nil {
			log.Printf("addrinfoToString: %s", err.Error())
			continue
		}
		log.Printf("getaddrinfo: resolved %s", addr)
		addrs = append(addrs, addr)
	}
	if len(addrs) < 1 {
		log.Printf("getaddrinfo: no address after ainfo loop")
		return nil, errors.New(DNSNoAnswerSuffix)
	}
	return addrs, nil
}

// errGetaddrinfoUnknownFamily indicates we don't know the address family.
var errGetaddrinfoUnknownFamily = errors.New("unknown address family")

// addrinfoToString is the function that converts a single entry
// in the struct_addrinfos linked list into a string.
//
// This function is adapted from cgoLookupIPCNAME
// https://github.com/golang/go/blob/go1.17.6/src/net/cgo_unix.go#L145
//
// SPDX-License-Identifier: BSD-3-Clause.
func (state *getaddrinfoState) addrinfoToString(r *C.struct_addrinfo) (string, error) {
	switch r.ai_family {
	case C.AF_INET:
		sa := (*syscall.RawSockaddrInet4)(unsafe.Pointer(r.ai_addr))
		addr := net.IPAddr{IP: state.copyIP(sa.Addr[:])}
		return addr.String(), nil
	case C.AF_INET6:
		sa := (*syscall.RawSockaddrInet6)(unsafe.Pointer(r.ai_addr))
		addr := net.IPAddr{
			IP:   state.copyIP(sa.Addr[:]),
			Zone: state.ifnametoindex(int(sa.Scope_id)),
		}
		return addr.String(), nil
	default:
		return "", errGetaddrinfoUnknownFamily
	}
}

// copyIP copies an net.IP.
//
// This function is adapted from copyIP
// https://github.com/golang/go/blob/go1.17.6/src/net/cgo_unix.go#L344
//
// SPDX-License-Identifier: BSD-3-Clause.
func (state *getaddrinfoState) copyIP(x net.IP) net.IP {
	if len(x) < 16 {
		return x.To16()
	}
	y := make(net.IP, len(x))
	copy(y, x)
	return y
}

// ifnametoindex converts an IPv6 scope index into an interface name.
//
// This function is adapted from ipv6ZoneCache.update
// https://github.com/golang/go/blob/go1.17.6/src/net/interface.go#L194
//
// SPDX-License-Identifier: BSD-3-Clause.
func (state *getaddrinfoState) ifnametoindex(idx int) string {
	iface, err := net.InterfaceByIndex(idx) // internally uses caching
	if err != nil {
		log.Printf("getadderinfo: InterfaceByIndex: %s", err.Error())
		return ""
	}
	return iface.Name
}
