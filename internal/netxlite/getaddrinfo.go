package netxlite

import (
	"context"
	"net"
)

// getaddrinfoLookupHost executes a DNS lookup using
// libc's getaddrinfo and returns the results.
//
// This function will attempt to return an ErrGetaddrinfo
// whenever the underlying getaddrinfo fails with one of
// the usual error codes, e.g., EAI_NONAME.
func getaddrinfoLookupHost(ctx context.Context, domain string) ([]string, error) {
	if !getaddrinfoAvailable() {
		return net.DefaultResolver.LookupHost(ctx, domain)
	}
	return getaddrinfoDoLookupHost(ctx, domain)
}

// ErrGetaddrinfo is the error returned by our getaddrinfo
// wrapper code. You should attempt to cast any DNS error using
// errors.As when you care about raw getaddrinfo errors.
type ErrGetaddrinfo struct {
	// Err is the error proper.
	Underlying error

	// Code is the original return code.
	Code int64
}

// newErrGetaddrinfo creates a new instance of the ErrGetaddrinfo type.
func newErrGetaddrinfo(code int64, err error) *ErrGetaddrinfo {
	return &ErrGetaddrinfo{
		Underlying: err,
		Code:       code,
	}
}

// Error returns the underlying error string.
func (err *ErrGetaddrinfo) Error() string {
	return err.Underlying.Error()
}

// Unwrap allows to get the underlying error.
func (err *ErrGetaddrinfo) Unwrap() error {
	return err.Underlying
}
