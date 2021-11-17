// Package errorsx contains error extensions.
package errorsx

import (
	"errors"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

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
			classifier = netxlite.ClassifyGenericError
		}
		err = &netxlite.ErrWrapper{
			Failure:    classifier(b.Error),
			Operation:  toOperationString(b.Error, b.Operation),
			WrappedErr: b.Error,
		}
	}
	return
}

func toOperationString(err error, operation string) string {
	var errwrapper *netxlite.ErrWrapper
	if errors.As(err, &errwrapper) {
		// Basically, as explained in ErrWrapper docs, let's
		// keep the child major operation, if any.
		if errwrapper.Operation == netxlite.ConnectOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == netxlite.HTTPRoundTripOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == netxlite.ResolveOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == netxlite.TLSHandshakeOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == netxlite.QUICHandshakeOperation {
			return errwrapper.Operation
		}
		if errwrapper.Operation == "quic_handshake_start" {
			return netxlite.QUICHandshakeOperation
		}
		if errwrapper.Operation == "quic_handshake_done" {
			return netxlite.QUICHandshakeOperation
		}
		// FALLTHROUGH
	}
	return operation
}
