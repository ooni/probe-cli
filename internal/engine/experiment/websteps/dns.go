package websteps

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type DNSConfig struct {
	Domain   string
	Resolver netxlite.ResolverLegacy
}

// DNSDo performs the DNS check.
func DNSDo(ctx context.Context, config DNSConfig) ([]string, error) {
	resolver := config.Resolver
	if resolver == nil {
		childResolver, err := netx.NewDNSClient(netx.Config{Logger: log.Log}, "doh://google")
		runtimex.PanicOnError(err, "NewDNSClient failed")
		resolver = childResolver
		resolver = &netxlite.ResolverIDNA{
			Resolver: netxlite.NewResolverLegacyAdapter(resolver),
		}
	}
	return resolver.LookupHost(ctx, config.Domain)
}
