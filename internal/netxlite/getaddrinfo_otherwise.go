//go:build !cgo

package netxlite

import (
	"context"
	"net"
)

// getaddrinfoDoLookupHost performs an host lookup with getaddrinfo.
func getaddrinfoDoLookupHost(ctx context.Context, domain string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, domain)
}
