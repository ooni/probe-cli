package netxlite

import "errors"

// ErrGetaddrinfo represents a getaddrinfo failure.
type ErrGetaddrinfo struct {
	// Err is the error proper.
	Underlying error

	// Code is getaddrinfo's return code.
	Code int64
}

// newErrGetaddrinfo creates a new instance of the ErrGetaddrinfo type.
func newErrGetaddrinfo(code int64, err error) *ErrGetaddrinfo {
	return &ErrGetaddrinfo{
		Underlying: err,
		Code:       code,
	}
}

// Error returns the underlying error's string.
func (err *ErrGetaddrinfo) Error() string {
	return err.Underlying.Error()
}

// Unwrap allows to get the underlying error value.
func (err *ErrGetaddrinfo) Unwrap() error {
	return err.Underlying
}

// ErrorToGetaddrinfoRetvalOrZero converts an arbitrary error to
// the return value of getaddrinfo. If err is nil or is not
// an instance of ErrGetaddrinfo, we just return zero.
func ErrorToGetaddrinfoRetvalOrZero(err error) int64 {
	var aierr *ErrGetaddrinfo
	if err != nil && errors.As(err, &aierr) {
		return aierr.Code
	}
	return 0
}
