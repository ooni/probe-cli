package netxlite

import (
	"context"
	"errors"
)

// getaddrinfoLookupHost attempts to execute a DNS lookup using
// libc's getaddrinfo and returns the results. If we do not link
// with libc, we'll fallback to net.DefaultResolver. Otherwise,
// in case getaddrinfo returns non-zero, we'll return an instance
// of ErrGetaddrinfo, which will contain the return code.
func getaddrinfoLookupHost(ctx context.Context, domain string) ([]string, error) {
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

// ErrorToGetaddrinfoRetval converts an arbitrary error to
// the return value of getaddrinfo. If there is no underlying
// getaddrinfo error, this function returns zero.
func ErrorToGetaddrinfoRetval(err error) int64 {
	if err != nil {
		var aierr *ErrGetaddrinfo
		if errors.As(err, &aierr) {
			return aierr.Code
		}
	}
	return 0
}
