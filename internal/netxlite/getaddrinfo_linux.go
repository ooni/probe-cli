// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build cgo && linux

package netxlite

/*
// Both glibc and musl expose the EAI_NODATA error if we
// ask them to expose it through this define. See below for
// more details on how each of the supported libcs hides
// (or does not hide) the EAI_NODATA define.
#cgo CFLAGS: -D_GNU_SOURCE
#include <netdb.h>
*/
import "C"

import (
	"runtime"
	"syscall"
)

// Implementation note: the original Go codebase separated linux and android
// but we want them to be in the same file, so we can implement tests for both
// operating system and increase our confidence that the behavior will be the
// one we'd like to see on Android systems.

var getaddrinfoAIFlags = getaddrinfoGetPlatformSpecificAIFlags(runtime.GOOS)

// This function returns the platforms-specific AI flags that go1.17.6
// used to set when we merged resolver's code into ooni/probe-cli
//
// SPDX-License-Identifier: BSD-3-Clause.
func getaddrinfoGetPlatformSpecificAIFlags(goos string) C.int {
	switch goos {
	case "android":
		return C.AI_CANONNAME
	default:
		// NOTE(rsc): In theory there are approximately balanced
		// arguments for and against including AI_ADDRCONFIG
		// in the flags (it includes IPv4 results only on IPv4 systems,
		// and similarly for IPv6), but in practice setting it causes
		// getaddrinfo to return the wrong canonical name on Linux.
		// So definitely leave it out.
		return C.AI_CANONNAME | C.AI_V4MAPPED | C.AI_ALL
	}
}

// Making constants available to Go code so we can run tests (it seems
// it's not possible to import C directly in tests, sadly).
const (
	aiCanonname = C.AI_CANONNAME
	aiV4Mapped  = C.AI_V4MAPPED
	aiAll       = C.AI_ALL
	eaiSystem   = C.EAI_SYSTEM
	eaiNoName   = C.EAI_NONAME
	eaiBadFlags = C.EAI_BADFLAGS
	eaiNoData   = C.EAI_NODATA
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
		return newErrGetaddrinfo(code, ErrOODNSNoSuchHost)
	case C.EAI_NODATA:
		return state.toErrorNODATA(err, goos)
	default:
		return newErrGetaddrinfo(code, ErrOODNSMisbehaving)
	}
}

// toErrorNODATA maps the EAI_NODATA value to the proper return value
// depending on the underlying operating system.
//
// As of 2022-05-28, this is the status of the major C libraries whose
// getaddrinfo return value we may end up processing here:
//
// 1. musl libc (statically linked Linux builds for official OONI
// Probe packages we build): EAI_NODATA is defined in netdb.h in a
// section guarded by _GNU_SOURCE and _BSD_SOURCE and the code
// does not otherwise ever use this definition.
//
// 2. GNU libc (which is what you would get if you compile OONI Probe
// for yourself in a GNU/Linux system): the codebase defines EAI_NODATA
// inside netdb.h protected by __USE_GNU, which is defined to 1 in
// include/features.h if the user defines _GNU_SOURCE. Additionally,
// the getaddrinfo implementation returns EAI_NODATA when a name
// exists but there's no associated address for such a name. There
// was a bug, fixed in glibc 2.27, were EAI_NONAME was returned
// when EAI_NODATA would actually have been more proper:
//
//	https://sourceware.org/bugzilla/show_bug.cgi?id=21922
//
// 3. Android libc: EAI_NODATA is defined in netdb.h and is not
// protected by any feature flag. The getaddrinfo function (as
// of 4ebdeebef74) calls android_getaddrinfofornet, which in turns
// calls android_getaddrinfofornetcontext. This function will
// eventually call android_getaddrinfo_proxy. If this function
// returns any status code different from EAI_SYSTEM, then bionic
// will return its return value. Otherwise, the code ends up
// calling explore_fqdn, which in turn calls nsdispatch, which
// is what NetBSD is still doing today.
//
// So, android_getaddrinfo_proxy was introduced a long time
// ago on October 28, 2010 by this commit:
//
//	https://github.com/aosp-mirror/platform_bionic/commit/a1dbf0b453801620565e5911f354f82706b0200d
//
// Then a subsequent commit changed android_getaddrinfo_proxy
// to basically default to EAI_NODATA on proxy errors:
//
//	https://github.com/aosp-mirror/platform_bionic/commit/c63e59039d28c352e3053bb81319e960c392dbd4
//
// As of today and 4ebdeebef74, android_getaddrinfo_proxy returns
// one of the following possible return codes:
//
// a) 0 on success;
//
// b) EAI_SYSTEM if it cannot speak to the proxy (which causes the code
// to fall through to the original NetBSD implementation);
//
// c) EAI_NODATA in all the other cases.
//
// The above discussion about Android provides us with a theory that explains the
// https://github.com/ooni/probe/issues/2029 issue. That said, we are still missing
// some bits, e.g., why some Android 6 phones did not experience this problem.
//
// We originally proposed to handle the EAI_NODATA error on Android like it was a
// EAI_NONAME error. However, this mapping seems very inaccurate. Any error inside
// the DNS proxy could cause EAI_NODATA (_unless_ we're "lucky" for some reason
// and the original NetBSD code runs). Therefore, the sanest choice is to introduce
// a new OONI error describing this error condition `android_dns_cache_no_data`
// and handle this error as a special case when checking for NXDOMAIN.
func (state *getaddrinfoState) toErrorNODATA(err error, goos string) error {
	switch goos {
	case "android":
		return newErrGetaddrinfo(C.EAI_NODATA, ErrAndroidDNSCacheNoData)
	default:
		return newErrGetaddrinfo(C.EAI_NODATA, ErrOODNSNoAnswer)
	}
}
