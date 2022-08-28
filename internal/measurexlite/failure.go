package measurexlite

import (
	"errors"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// NewFailure creates an OONI failure from an error. If the error is nil,
// we return nil. If the error is not already an ErrWrapper, it's converted
// to an ErrWrapper. If the ErrWrapper's Failure is not empty, we return its
// string representation. Otherwise we return a string indicating that
// an ErrWrapper has an empty failure (should not happen).
//
// See https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md
// for more information about OONI failures.
func NewFailure(err error) *string {
	if err == nil {
		return nil
	}
	// The following code guarantees that the error is always wrapped even
	// when we could not actually hit our code that does the wrapping. A case
	// in which this could happen is with context deadline for HTTP when you
	// have wrapped the underlying dialers but not the Transport.
	var errWrapper *netxlite.ErrWrapper
	if !errors.As(err, &errWrapper) {
		err := netxlite.NewTopLevelGenericErrWrapper(err)
		couldConvert := errors.As(err, &errWrapper)
		runtimex.PanicIfFalse(couldConvert, "we should have an ErrWrapper here")
	}
	s := errWrapper.Failure
	if s == "" {
		s = "unknown_failure: errWrapper.Failure is empty"
	}
	return &s
}
