package errorsx

import "encoding/json"

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
	// New code will always nest ErrWrapper and you need to
	// walk the chain to find what happened.
	//
	// The following comment describes the DEPRECATED
	// legacy behavior implements by internal/engine/legacy/errorsx:
	//
	// If possible, the Operation string
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

// MarshalJSON converts an ErrWrapper to a JSON value.
func (e *ErrWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Failure)
}

// Classifier is the type of function that performs classification.
type Classifier func(err error) string

// NewErrWrapper creates a new ErrWrapper using the given
// classifier, operation name, and underlying error.
//
// This function panics if classifier is nil, or operation
// is the empty string or error is nil.
func NewErrWrapper(c Classifier, op string, err error) *ErrWrapper {
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
