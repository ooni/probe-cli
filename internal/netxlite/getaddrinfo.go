package netxlite

import (
	"context"
	"errors"
)

// getaddrinfoLookupHost performs a DNS lookup and returns the
// results. If we were compiled with CGO_ENABLED=0, then this
// function calls net.DefaultResolver.LookupHost. Otherwise,
// we call getaddrinfo. In such a case, if getaddrinfo returns a nonzero
// return value, we'll return as error an instance of the
// ErrGetaddrinfo error. This error will contain the specific
// code returned by getaddrinfo in its .Code field.
func getaddrinfoLookupHost(ctx context.Context, domain string) ([]string, error) {
	return getaddrinfoDoLookupHost(ctx, domain)
}

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

// ErrorToGetaddrinfoRetval converts an arbitrary error to
// the return value of getaddrinfo. If err is nil or is not
// an instance of ErrGetaddrinfo, we just return zero.
func ErrorToGetaddrinfoRetval(err error) int64 {
	var aierr *ErrGetaddrinfo
	if err != nil && errors.As(err, &aierr) {
		return aierr.Code
	}
	return 0
}
