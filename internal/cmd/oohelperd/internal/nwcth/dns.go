package nwcth

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// newResolver creates a new DNS resolver instance
func newResolver() netxlite.Resolver {
	childResolver, err := netx.NewDNSClient(netx.Config{Logger: log.Log}, "doh://google")
	runtimex.PanicOnError(err, "NewDNSClient failed")
	var r netxlite.Resolver = childResolver
	r = &netxlite.IDNAResolver{Resolver: r}
	return r
}

// DNSDo performs the DNS check.
func DNSDo(ctx context.Context, domain string, resolver netxlite.Resolver) ([]string, error) {
	return resolver.LookupHost(ctx, domain)
}
