package netxlite

import "errors"

// Name of the resolver we use when we link with libc and use getaddrinfo directly.
//
// See https://github.com/ooni/spec/pull/257 for more info.
const StdlibResolverGetaddrinfo = "getaddrinfo"

// Name of the resolver we use when we don't link with libc and use net.Resolver.
//
// See https://github.com/ooni/spec/pull/257 for more info.
const StdlibResolverGolangNetResolver = "golang_net_resolver"

// Legacy name of the resolver we use when we're don't know whether we're using
// getaddrinfo, but we're using net.Resolver, and we're splitting the answer
// in two A and AAAA queries. Eventually will become deprecated.
//
// See https://github.com/ooni/spec/pull/257 for more info.
const StdlibResolverSystem = "system"

// ErrGetaddrinfo represents a getaddrinfo failure.
type ErrGetaddrinfo struct {
	// Err is the error proper.
	Underlying error

	// Code is getaddrinfo's return code.
	Code int64
}

// NewErrGetaddrinfo creates a new instance of the ErrGetaddrinfo type.
func NewErrGetaddrinfo(code int64, err error) *ErrGetaddrinfo {
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
