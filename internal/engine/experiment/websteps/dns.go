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
	Resolver netxlite.Resolver
}

// DNSDo performs the DNS check.
func DNSDo(ctx context.Context, config DNSConfig) ([]string, error) {
	if config.Resolver == nil {
		childResolver, err := netx.NewDNSClient(netx.Config{Logger: log.Log}, "doh://google")
		runtimex.PanicOnError(err, "NewDNSClient failed")
		var r netxlite.Resolver = childResolver
		r = &netxlite.IDNAResolver{Resolver: r}
		config.Resolver = r
	}
	return config.Resolver.LookupHost(ctx, config.Domain)
}
