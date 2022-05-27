//go:build !cgo

package netxlite

import (
	"context"
	"net"
)

func getaddrinfoLookupANY(ctx context.Context, domain string) ([]string, string, error) {
	return net.DefaultResolver.LookupHost(ctx, domain)
}
