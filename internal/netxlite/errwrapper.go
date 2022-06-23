package netxlite

import (
	"encoding/json"
	"errors"
)

// ErrWrapper is our error wrapper for Go errors. The key objective of
// this structure is to properly set Failure, which is also returned by
// the Error() method, to be one of the OONI failure strings.
//
// OONI failure strings are defined in the github.com/ooni/spec repo
// at https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md.
type ErrWrapper struct {
	// Failure is the OONI failure string. The failure strings are
	// loosely backward compatible with Measurement Kit.
	//
	// This is either one of the FailureXXX strings or any other
	// string like `unknown_failure: ...`. The latter represents an
	// error that we have not yet mapped to a failure.
	Failure string

	// Operation is the operation that failed.
	//
	// If possible, the Operation string SHOULD be a _major_
	// operation. Major operations are:
	//
	// - ResolveOperation: resolving a domain name failed
	// - ConnectOperation: connecting to an IP failed
	// - TLSHandshakeOperation: TLS handshaking failed
	// - QUICHandshakeOperation: QUIC handshaking failed
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

// Error returns the OONI failure string for this error.
func (e *ErrWrapper) Error() string {
	return e.Failure
}

// Unwrap allows to access the underlying error.
func (e *ErrWrapper) Unwrap() error {
	return e.WrappedErr
}

// MarshalJSON converts an ErrWrapper to a JSON value.
func (e *ErrWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Failure)
}

// classifier is the type of the function that maps a Go error
// to a OONI failure string defined at
// https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md.
type classifier func(err error) string

// NewErrWrapper creates a new ErrWrapper using the given
// classifier, operation name, and underlying error.
//
// This function panics if classifier is nil, or operation
// is the empty string or error is nil.
//
// If the err argument has already been classified, the returned
// error wrapper will use the same classification string and
// will determine whether to keep the major operation as documented
// in the ErrWrapper.Operation documentation.
func NewErrWrapper(c classifier, op string, err error) *ErrWrapper {
	var wrapper *ErrWrapper
	if errors.As(err, &wrapper) {
		return &ErrWrapper{
			Failure:    wrapper.Failure,
			Operation:  classifyOperation(wrapper, op),
			WrappedErr: err,
		}
	}
	if c == nil {
		panic("nil classifier")
	}
	if op == "" {
		panic("empty op")
	}
	if err == nil {
		panic("nil err")
	}
	return &ErrWrapper{
		Failure:    c(err),
		Operation:  op,
		WrappedErr: err,
	}
}

// TODO(https://github.com/ooni/probe/issues/2163): we can really
// simplify the error wrapping situation here by just dropping
// NewErrWrapper and always using MaybeNewErrWrapper.

// MaybeNewErrWrapper is like NewErrWrapper except that this
// function won't panic if passed a nil error.
func MaybeNewErrWrapper(c classifier, op string, err error) error {
	if err != nil {
		return NewErrWrapper(c, op, err)
	}
	return nil
}

// NewTopLevelGenericErrWrapper wraps an error occurring at top
// level using a generic classifier as classifier. This is the
// function you should call when you suspect a given error hasn't
// already been wrapped. This function panics if err is nil.
//
// If the err argument has already been classified, the returned
// error wrapper will use the same classification string and
// failed operation of the original error.
func NewTopLevelGenericErrWrapper(err error) *ErrWrapper {
	return NewErrWrapper(ClassifyGenericError, TopLevelOperation, err)
}

func classifyOperation(ew *ErrWrapper, operation string) string {
	// Basically, as explained in ErrWrapper docs, let's
	// keep the child major operation, if any.
	//
	// QUIRK: this code is legacy code and we should not change
	// it unless we also change the experiments that depend on it
	// for determining the blocking reason based on the failed
	// operation value (e.g., telegram, web connectivity).
	if ew.Operation == ConnectOperation {
		return ew.Operation
	}
	if ew.Operation == HTTPRoundTripOperation {
		return ew.Operation
	}
	if ew.Operation == ResolveOperation {
		return ew.Operation
	}
	if ew.Operation == TLSHandshakeOperation {
		return ew.Operation
	}
	if ew.Operation == QUICHandshakeOperation {
		return ew.Operation
	}
	if ew.Operation == "quic_handshake_start" {
		return QUICHandshakeOperation
	}
	if ew.Operation == "quic_handshake_done" {
		return QUICHandshakeOperation
	}
	return operation
}
