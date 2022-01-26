package netxlite

import (
	"errors"
	"strings"
)

// This file contains weird stuff that we carried over from
// the original netx implementation and that we cannot remove
// or change without thinking about the consequences.

// See https://github.com/ooni/probe/issues/1985
var errReduceErrorsEmptyList = errors.New("bug: reduceErrors given an empty list")

// quirkReduceErrors finds a known error in a list of errors since
// it's probably most relevant. If this error is not found, just
// return the first error according to this reasoning:
//
// If we have a known error, let's consider this the real error
// since it's probably most relevant. Otherwise let's return the
// first considering that (1) local resolvers likely will give
// us IPv4 first and (2) also our resolver does that. So, in case
// the user has no IPv6 connectivity, an IPv6 error is going to
// appear later in the list of errors.
//
// Honestly, the above reasoning does not feel very solid and
// we also have an IMPLICIT assumption on our resolver returning
// IPv4 before IPv6 _which is a really fragile one_. We try to
// remediate with quirkSortIPAddrs (see below).
//
// This is CLEARLY a QUIRK anyway. There may code depending on how
// we do things here and it's tricky to remove this behavior.
//
// See TODO(https://github.com/ooni/probe/issues/1779).
func quirkReduceErrors(errorslist []error) error {
	if len(errorslist) == 0 {
		// See https://github.com/ooni/probe/issues/1985
		return errReduceErrorsEmptyList
	}
	for _, err := range errorslist {
		var wrapper *ErrWrapper
		if errors.As(err, &wrapper) && !strings.HasPrefix(
			err.Error(), "unknown_failure",
		) {
			return err
		}
	}
	return errorslist[0]
}

// quirkSortIPAddrs sorts IP addresses so that IPv4 appears
// before IPv6. Dialers SHOULD call this code.
//
// It saddens me to have this quirk, but it is here to pair
// with quirkReduceErrors, which assumes that <facepalm>.
//
// See TODO(https://github.com/ooni/probe/issues/1779).
func quirkSortIPAddrs(addrs []string) (out []string) {
	isIPv6 := func(x string) bool {
		// This check for identifying IPv6 is discussed
		// at https://stackoverflow.com/questions/22751035
		// and seems good-enough for our purposes.
		return strings.Contains(x, ":")
	}
	isIPv4 := func(x string) bool {
		return !isIPv6(x)
	}
	for _, addr := range addrs {
		if isIPv4(addr) {
			out = append(out, addr)
		}
	}
	for _, addr := range addrs {
		if isIPv6(addr) {
			out = append(out, addr)
		}
	}
	return
}
