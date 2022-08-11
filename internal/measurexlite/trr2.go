package measurexlite

//
// Trusted Recursive Resolver
//

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewTrustedRecursiveResolver2 returns a new trusted recursive resolver
// When the URL is empty, we use a reliable resolver URL as the default resolver
// When the timeout is 0, we use Firefox's default request timeout value
// TODO(DecFox): Maybe we can be more liberal with the default timeout value
func NewTrustedRecursiveResolver2(logger model.Logger, URL string, timeout int) model.SimpleResolver {
	const mozillaURL = "https://mozilla.cloudflare-dns.com/dns-query"
	if URL == "" {
		URL = mozillaURL
	}
	if timeout == 0 {
		timeout = 1500
	}
	return &TrustedRecursiveResolver2{
		Logger:                            logger,
		URL:                               URL,
		NewParallelDNSOverHTTPSResolverFn: nil,
		ResolverSystem:                    netxlite.NewStdlibResolver(logger),
		Timeout:                           time.Duration(timeout * int(time.Millisecond)),
	}
}

// TrustedRecursiveResolver2 emulates Firefox's TRR2 mode.
// TODO(DecFox): Since TraustedRecursiveResolver2 is a model.SimpleResolver,
// and we only make use of Lookuphost, the ResolverSystem and DoH factory
// fields could also be replaced with a simple resolver
type TrustedRecursiveResolver2 struct {
	// Logger is MANDATORY and is passed to the ParallelDNSOverHTTPSResolver
	Logger model.Logger

	// URL is MANDATORY and is forwarded as the url to be used in the DoH resolver
	URL string

	// NewParallelDNSOverHTTPSResolverFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewParallelDNSOverHTTPSResolver factory
	NewParallelDNSOverHTTPSResolverFn func() model.Resolver

	// ResolverSystem is MANDATORY and is used as a fallback if the DoH resolver fails
	ResolverSystem model.Resolver

	// Timeout is MANDATORY and a configurable timeout for the DoH resolver
	Timeout time.Duration
}

var _ model.SimpleResolver = &TrustedRecursiveResolver2{}

// Network implements model.Resolver.Network
func (trr *TrustedRecursiveResolver2) Network() string {
	return "trr2"
}

// LookupHost implements model.SimpleResolver.LookupHost
func (trr *TrustedRecursiveResolver2) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	reso := trr.newParallelDNSOverHTTPSResolver(trr.Logger, trr.URL)
	c, cancel := context.WithTimeout(ctx, trr.Timeout)
	defer cancel()
	addrs, err := reso.LookupHost(c, hostname)
	if err != nil {
		addrs, err = trr.ResolverSystem.LookupHost(ctx, hostname)
	}
	return addrs, err
}

// newParallelDNSOverHTTPSResolver is a factory that returns a parallel DoH resolver
func (trr *TrustedRecursiveResolver2) newParallelDNSOverHTTPSResolver(logger model.Logger, URL string) model.SimpleResolver {
	if trr.NewParallelDNSOverHTTPSResolverFn != nil {
		return trr.NewParallelDNSOverHTTPSResolverFn()
	}
	return netxlite.NewParallelDNSOverHTTPSResolver(logger, URL)
}
