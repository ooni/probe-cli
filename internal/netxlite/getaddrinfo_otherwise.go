//go:build !cgo

package netxlite

import (
	"context"
	"net"
)

// getaddrinfoResolverNetwork returns the "network" that is actually
// been used to implement the getaddrinfo resolver.
//
// This is the CGO_ENABLED=0 implementation of this function, which
// always returns the string [StdlibResolverGolangNetResolver], because in this scenario
// we are actually using whatever resolver is used under the hood by the stdlib.
//
// See https://github.com/ooni/probe/issues/2029#issuecomment-1140805266
// for an explanation of why calling this resolver "netgo" was wrong.
//
// See https://github.com/ooni/spec/pull/257 for additional documentation
// regarding using "golang_net_resolver" instead of "go".
func getaddrinfoResolverNetwork() string {
	return StdlibResolverGolangNetResolver
}

// getaddrinfoLookupANY attempts to perform an ANY lookup using getaddrinfo.
//
// This is the CGO_ENABLED=0 implementation of this function.
//
// Arguments:
//
// - ctx is the context for deadline/timeout/cancellation
//
// - domain is the domain to lookup
//
// This function returns the list of looked up addresses, an always-empty
// CNAME, and the error that occurred. On error, the list of addresses is empty.
func getaddrinfoLookupANY(ctx context.Context, domain string) ([]string, string, error) {
	al, err := net.DefaultResolver.LookupHost(ctx, domain)
	return al, "", err
}
