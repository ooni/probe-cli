package measurexlite

//
// Trusted Recursive Resolver
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewTrustedRecursiveResolver2 returns a new trusted recursive resolver
// When the URL is empty, we use a reliable resolver URL as the default resolver
// When the timeout is 0, we use Firefox's default request timeout value
// TODO(DecFox): Maybe we can be more liberal with the default timeout value
func NewTrustedRecursiveResolver2(logger model.Logger, URL string, timeout int) model.Resolver {
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
type TrustedRecursiveResolver2 struct {
	// Logger is MANDATORY and is passed to the ParallelDNSOverHTTPSResolver
	Logger model.Logger

	// URL is MANDATORY and is forwarded as the url to be used in the DoH resolver
	URL string

	// NewParallelDNSOverHTTPSResolverFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewParallelDNSOverHTTPSResolver factory
	NewParallelDNSOverHTTPSResolverFn func(model.Logger, string) model.Resolver

	// ResolverSystem is MANDATORY and is used as a fallback if the DoH resolver fails
	ResolverSystem model.Resolver

	// Timeout is OPTIONAL and a configurable timeout for the DoH resolver
	Timeout time.Duration
}

var _ model.Resolver = &TrustedRecursiveResolver2{}

// Address implements model.Resolver.Address
func (trr *TrustedRecursiveResolver2) Address() string {
	return trr.URL
}

// Network implements model.Resolver.Network
func (trr *TrustedRecursiveResolver2) Network() string {
	return "trr2"
}

// CloseIdleConnections implements model.Resolver.CloseIdleConnections
func (trr TrustedRecursiveResolver2) CloseIdleConnections() {
	trr.ResolverSystem.CloseIdleConnections()
}

// LookupHost implements model.Resolver.LookupHost
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

// LookupHTTPS implements model.Resolver.LookupHTTPS
func (trr *TrustedRecursiveResolver2) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	reso := trr.newParallelDNSOverHTTPSResolver(trr.Logger, trr.URL)
	c, cancel := context.WithTimeout(ctx, trr.Timeout)
	defer cancel()
	out, err := reso.LookupHTTPS(c, domain)
	if err != nil {
		out, err = trr.ResolverSystem.LookupHTTPS(ctx, domain)
	}
	return out, err
}

// LookupNS implements model.Resolver.LookupNS
func (trr *TrustedRecursiveResolver2) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	reso := trr.newParallelDNSOverHTTPSResolver(trr.Logger, trr.URL)
	c, cancel := context.WithTimeout(ctx, trr.Timeout)
	defer cancel()
	out, err := reso.LookupNS(c, domain)
	if err != nil {
		out, err = trr.ResolverSystem.LookupNS(ctx, domain)
	}
	return out, err
}

// newParallelDNSOverHTTPSResolver is a factory that returns a parallel DoH resolver
func (trr *TrustedRecursiveResolver2) newParallelDNSOverHTTPSResolver(logger model.Logger, URL string) model.Resolver {
	if trr.NewParallelDNSOverHTTPSResolverFn != nil {
		return trr.NewParallelDNSOverHTTPSResolverFn(logger, URL)
	}
	return netxlite.NewParallelDNSOverHTTPSResolver(logger, URL)
}
