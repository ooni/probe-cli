//go:build: cgo

package netxlite

/*
// On Unix systems, getaddrinfo is part of libc. On Windows,
// instead, we need to explicitly link with winsock2.
#cgo windows LDFLAGS: -lws2_32

#ifndef _WIN32
#include <netdb.h> // for getaddrinfo
#include <sys/socket.h> // for C.SOCK_STREAM and C.AF_*
#else
#include <ws2tcpip.h> // for getaddrinfo
#endif
*/
import "C"

import (
	"context"
	"errors"
	"net"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/miekg/dns"
)

// getaddrinfoResolverNetwork returns the "network" that is actually
// been used to implement the getaddrinfo resolver.
//
// This is the CGO_ENABLED=1 implementation of this function, which
// always returns the string [StdlibResolverGetaddrinfo], because in this scenario
// we are actually calling the getaddrinfo libc function.
//
// See https://github.com/ooni/spec/pull/257 for more information on how
// we evolved our naming of the "stdlib" resolver over time.
func getaddrinfoResolverNetwork() string {
	return StdlibResolverGetaddrinfo
}

// getaddrinfoLookupANY attempts to perform an ANY lookup using getaddrinfo.
//
// This is the CGO_ENABLED=1 implementation of this function.
//
// Arguments:
//
// - ctx is the context for deadline/timeout/cancellation
//
// - domain is the domain to lookup
//
// This function returns the list of looked up addresses, the CNAME, and
// the error that occurred. On error, the list of addresses is empty. The
// CNAME may be empty on success, if there's no CNAME, but may also be
// non-empty on failure, if the lookup result included a CNAME answer but
// did not include any A or AAAA answers. If getaddrinfo returns a nonzero
// return value, we'll return as error an instance of the
// ErrGetaddrinfo error. This error will contain the specific
// code returned by getaddrinfo in its .Code field.
func getaddrinfoLookupANY(ctx context.Context, domain string) ([]string, string, error) {
	return getaddrinfoStateSingleton.LookupANY(ctx, domain)
}

// getaddrinfoSingleton is the getaddrinfo singleton.
var getaddrinfoStateSingleton = newGetaddrinfoState(getaddrinfoNumSlots)

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

	// lookupANY is the function that actually implements
	// the lookup ANY lookup using getaddrinfo.
	lookupANY func(domain string) ([]string, string, error)
}

// getaddrinfoNumSlots is the maximum number of parallel calls
// to getaddrinfo we may have at any given time.
const getaddrinfoNumSlots = 8

// newGetaddrinfoState creates the getaddrinfo state.
func newGetaddrinfoState(numSlots int) *getaddrinfoState {
	state := &getaddrinfoState{
		sema:      make(chan *getaddrinfoSlot, numSlots),
		lookupANY: nil,
	}
	state.lookupANY = state.doLookupANY
	return state
}

// lookupANY invokes getaddrinfo and returns the results.
func (state *getaddrinfoState) LookupANY(ctx context.Context, domain string) ([]string, string, error) {
	if err := state.grabSlot(ctx); err != nil {
		return nil, "", err
	}
	defer state.releaseSlot()
	return state.doLookupANY(domain)
}

// grabSlot grabs a slot for calling getaddrinfo. This function may block until
// a slot becomes available (or until the context is done).
func (state *getaddrinfoState) grabSlot(ctx context.Context) error {
	// Implementation note: the channel has getaddrinfoNumSlots capacity, hence
	// the first getaddrinfoNumSlots channel writes will succeed and all the
	// subsequent ones will block. To unblock a pending request, we release a
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

// doLookupANY calls getaddrinfo. We assume that you've already grabbed a
// slot and you're defer-releasing it when you're done.
//
// This function is adapted from cgoLookupIPCNAME
// https://github.com/golang/go/blob/go1.17.6/src/net/cgo_unix.go#L145
//
// SPDX-License-Identifier: BSD-3-Clause.
func (state *getaddrinfoState) doLookupANY(domain string) ([]string, string, error) {
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
		return nil, "", state.toError(int64(code), err, runtime.GOOS)
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
func (state *getaddrinfoState) toAddressList(res *C.struct_addrinfo) ([]string, string, error) {
	var (
		addrs     []string
		canonname string
	)
	for r := res; r != nil; r = r.ai_next {
		if r.ai_canonname != nil {
			// See https://github.com/ooni/probe/issues/2293
			canonname = dns.CanonicalName(C.GoString(r.ai_canonname))
		}
		// We only asked for SOCK_STREAM, but check anyhow.
		if r.ai_socktype != C.SOCK_STREAM {
			continue
		}
		addr, err := state.addrinfoToString(r)
		if err != nil {
			continue
		}
		addrs = append(addrs, addr)
	}
	if len(addrs) < 1 {
		return nil, canonname, ErrOODNSNoAnswer
	}
	return addrs, canonname, nil
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
		addr := net.IPAddr{IP: getaddrinfoCopyIP(sa.Addr[:])}
		return addr.String(), nil
	case C.AF_INET6:
		sa := (*syscall.RawSockaddrInet6)(unsafe.Pointer(r.ai_addr))
		addr := net.IPAddr{
			IP:   getaddrinfoCopyIP(sa.Addr[:]),
			Zone: getaddrinfoIfNametoindex(int(sa.Scope_id)),
		}
		return addr.String(), nil
	default:
		return "", errGetaddrinfoUnknownFamily
	}
}

// getaddrinfoCopyIP copies a net.IP.
//
// This function is adapted from copyIP
// https://github.com/golang/go/blob/go1.17.6/src/net/cgo_unix.go#L344
//
// SPDX-License-Identifier: BSD-3-Clause.
func getaddrinfoCopyIP(x net.IP) net.IP {
	if len(x) < 16 {
		return x.To16()
	}
	y := make(net.IP, len(x))
	copy(y, x)
	return y
}

// getaddrinfoIfNametotindex converts an IPv6 scope index into an interface name.
//
// This function is adapted from ipv6ZoneCache.update
// https://github.com/golang/go/blob/go1.17.6/src/net/interface.go#L194
//
// SPDX-License-Identifier: BSD-3-Clause.
func getaddrinfoIfNametoindex(idx int) string {
	iface, err := net.InterfaceByIndex(idx) // internally uses caching
	if err != nil {
		return ""
	}
	return iface.Name
}
