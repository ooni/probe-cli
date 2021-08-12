package nwcth

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// newResolver creates a new DNS resolver instance
func newResolver() netxlite.Resolver {
	// TODO(bassosimone,kelmenhorst): what complexity do we need here for the resolver? is this enough?
	return &netxlite.ResolverSystem{}
}

// DNSDo performs the DNS check.
func DNSDo(ctx context.Context, domain string, resolver netxlite.Resolver) ([]string, error) {
	return resolver.LookupHost(ctx, domain)
}
