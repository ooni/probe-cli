// Package errorsx contains error extensions.
package errorsx

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/scrubber"
)

// ErrDNSBogon indicates that we found a bogon address. This is the
// correct value with which to initialize MeasurementRoot.ErrDNSBogon
// to tell this library to return an error when a bogon is found.
var ErrDNSBogon = errors.New("dns: detected bogon address")

// ErrWrapper is our error wrapper for Go errors. The key objective of
// this structure is to properly set Failure, which is also returned by
// the Error() method, so be one of the OONI defined strings.
type ErrWrapper struct {
	// Failure is the OONI failure string. The failure strings are
	// loosely backward compatible with Measurement Kit.
	//
	// This is either one of the FailureXXX strings or any other
	// string like `unknown_failure ...`. The latter represents an
	// error that we have not yet mapped to a failure.
	Failure string

	// Operation is the operation that failed. If possible, it
	// SHOULD be a _major_ operation. Major operations are:
	//
	// - ResolveOperation: resolving a domain name failed
	// - ConnectOperation: connecting to an IP failed
	// - TLSHandshakeOperation: TLS handshaking failed
	// - HTTPRoundTripOperation: other errors during round trip
	//
	// Because a network connection doesn't necessarily know
	// what is the current major operation we also have the
	// following _minor_ operations:
	//
	// - CloseOperation: CLOSE failed
	// - ReadOperation: READ failed
	// - WriteOperation: WRITE failed
	//
	// If an ErrWrapper referring to a major operation is wrapping
	// another ErrWrapper and such ErrWrapper already refers to
	// a major operation, then the new ErrWrapper should use the
	// child ErrWrapper major operation. Otherwise, it should use
	// its own major operation. This way, the topmost wrapper is
	// supposed to refer to the major operation that failed.
	Operation string

	// WrappedErr is the error that we're wrapping.
	WrappedErr error
}

// Error returns a description of the error that occurred.
func (e *ErrWrapper) Error() string {
	return e.Failure
}

// Unwrap allows to access the underlying error
func (e *ErrWrapper) Unwrap() error {
	return e.WrappedErr
}

// SafeErrWrapperBuilder contains a builder for ErrWrapper that
// is safe, i.e., behaves correctly when the error is nil.
type SafeErrWrapperBuilder struct {
	// Error is the error, if any
	Error error

	// Classifier is the local error to string classifier. When there is no
	// configured classifier we will use the generic classifier.
	Classifier func(err error) string

	// Operation is the operation that failed
	Operation string
}

// MaybeBuild builds a new ErrWrapper, if b.Error is not nil, and returns
// a nil error value, instead, if b.Error is nil.
func (b SafeErrWrapperBuilder) MaybeBuild() (err error) {
	if b.Error != nil {
		classifier := b.Classifier
		if classifier == nil {
			classifier = toFailureString
		}
		err = &ErrWrapper{
			Failure:    classifier(b.Error),
			Operation:  toOperationString(b.Error, b.Operation),
			WrappedErr: b.Error,
		}
	}
	return
}

// TODO (kelmenhorst, bassosimone):
// Use errors.Is / errors.As more often, when possible, in this classifier.
// These methods are more robust to library changes than strings.
// errors.Is / errors.As can only be used when the error is exported.
func toFailureString(err error) string {
	// The list returned here matches the values used by MK unless
	// explicitly noted otherwise with a comment.

	// TODO(bassosimone): we need to always apply this rule not only here
	// when we're making the most generic conversion.
	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		return errwrapper.Error() // we've already wrapped it
	}

	if failure := toSyscallErr(err); failure != "" {
		return failure
	}

	if errors.Is(err, context.Canceled) {
		return FailureInterrupted
	}
	s := err.Error()
	if strings.HasSuffix(s, "operation was canceled") {
		return FailureInterrupted
	}
	if strings.HasSuffix(s, "EOF") {
		return FailureEOFError
	}
	if strings.HasSuffix(s, "context deadline exceeded") {
		return FailureGenericTimeoutError
	}
	if strings.HasSuffix(s, "transaction is timed out") {
		return FailureGenericTimeoutError
	}
	if strings.HasSuffix(s, "i/o timeout") {
		return FailureGenericTimeoutError
	}
	// TODO(kelmenhorst,bassosimone): this can probably be moved since it's TLS specific
	if strings.HasSuffix(s, "TLS handshake timeout") {
		return FailureGenericTimeoutError
	}
	if strings.HasSuffix(s, "no such host") {
		// This is dns_lookup_error in MK but such error is used as a
		// generic "hey, the lookup failed" error. Instead, this error
		// that we return here is significantly more specific.
		return FailureDNSNXDOMAINError
	}
	formatted := fmt.Sprintf("unknown_failure: %s", s)
	return scrubber.Scrub(formatted) // scrub IP addresses in the error
}

func toOperationString(err error, operation string) string {
	var errwrapper *ErrWrapper
	if errors.As(err, &errwrapper) {
		// Basically, as explained in ErrWrapper docs, let's
		// keep the child major operation, if any.
		if errwrapper.Operation == ConnectOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == HTTPRoundTripOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == ResolveOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == TLSHandshakeOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == QUICHandshakeOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == "quic_handshake_start" {
			return QUICHandshakeOperation
		}
		if errwrapper.Operation == "quic_handshake_done" {
			return QUICHandshakeOperation
		}
		// FALLTHROUGH
	}
	return operation
}
