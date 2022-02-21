//go:build !cgo

package netxlite

import (
	"context"
	"errors"
)

// getaddrinfoAvailable returns whether getaddrinfo is available.
func getaddrinfoAvailable() bool {
	return false
}

// errGetaddrinfoNotAvailable means that getaddrinfo is not available.
var errGetaddrinfoNotAvailable = errors.New("getaddrinfo: not available")

// getaddrinfoDoLookupHost performs an host lookup with getaddrinfo.
func getaddrinfoDoLookupHost(ctx context.Context, domain string) ([]string, error) {
	return nil, errGetaddrinfoNotAvailable
}
