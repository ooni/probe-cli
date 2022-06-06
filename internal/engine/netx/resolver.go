package netx

//
// Resolver from Config.
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewResolver creates a new resolver from the specified config.
func NewResolver(config Config) model.Resolver {
	if config.BaseResolver == nil {
		config.BaseResolver = netxlite.NewResolverSystem()
	}
	r := netxlite.WrapResolver(
		model.ValidLoggerOrDefault(config.Logger),
		config.BaseResolver,
	)
	r = netxlite.MaybeWrapWithCachingResolver(config.CacheResolutions, r)
	r = netxlite.MaybeWrapWithStaticDNSCache(config.DNSCache, r)
	r = netxlite.MaybeWrapWithBogonResolver(config.BogonIsError, r)
	return config.Saver.WrapResolver(r) // WAI when config.Saver==nil
}
