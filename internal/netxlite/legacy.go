package netxlite

import (
	"errors"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
)

// reduceErrors finds a known error in a list of errors since
// it's probably most relevant.
//
// Deprecation warning
//
// Albeit still used, this function is now DEPRECATED.
//
// In perspective, we would like to transition to a scenario where
// full dialing is NOT used for measurements and we return a multierror here.
func reduceErrors(errorslist []error) error {
	if len(errorslist) == 0 {
		return nil
	}
	// If we have a known error, let's consider this the real error
	// since it's probably most relevant. Otherwise let's return the
	// first considering that (1) local resolvers likely will give
	// us IPv4 first and (2) also our resolver does that. So, in case
	// the user has no IPv6 connectivity, an IPv6 error is going to
	// appear later in the list of errors.
	for _, err := range errorslist {
		var wrapper *errorx.ErrWrapper
		if errors.As(err, &wrapper) && !strings.HasPrefix(
			err.Error(), "unknown_failure",
		) {
			return err
		}
	}
	// TODO(bassosimone): handle this case in a better way
	return errorslist[0]
}
