//go:build !cgo

package netxlite

import (
	"context"
	"net"
)

func getaddrinfoDoLookupHost(ctx context.Context, domain string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, domain)
}
