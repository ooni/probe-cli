package measurexlite

//
// Trusted Recursive Resolver
//

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewTrustedRecursiveResolver2 returns a new trusted recursive resolver
func NewTrustedRecursiveResolver2(logger model.Logger, URL string) model.Resolver {
	const mozillaURL = "https://mozilla.cloudflare-dns.com/dns-query"
	if URL == "" {
		URL = mozillaURL
	}
	return &TrustedRecursiveResolver2{
		Logger:                            logger,
		URL:                               URL,
		NewParallelDNSOverHTTPSResolverFn: nil,
		ResolverSystem:                    netxlite.NewStdlibResolver(logger),
	}
}

// TrustedRecursiveResolver2 emulates Firefox's TRR2 mode
type TrustedRecursiveResolver2 struct {
	Logger model.Logger
	URL    string

	// NewParallelDNSOverHTTPSResolverFn is OPTIONAL and can be used to overide
	// calls to the netxlite.NewParallelDNSOverHTTPSResolver factory
	NewParallelDNSOverHTTPSResolverFn func(model.Logger, string) model.Resolver

	// ResolverSystem is MANADATORY and is used as a fallback if the DoH resolver fails
	ResolverSystem model.Resolver
}

var _ model.Resolver = &TrustedRecursiveResolver2{}

// Address implements model.Resolver.Address
// we can't return a decisive address here since we don't know if LookupHost
// succeeds with the DoH resolver or the system resolver
func (trr *TrustedRecursiveResolver2) Address() string {
	return ""
}

// Network implements model.Resolver.Network
// we can't return a decisive network here since we don't know if LookupHost
// succeeds with the DoH resolver or the system resolver
func (trr *TrustedRecursiveResolver2) Network() string {
	return ""
}

// CloseIdleConnections implements model.Resolver.CloseIdleConnections
// we only check the fallback resolver since DoH resolver is generated on the fly
func (trr TrustedRecursiveResolver2) CloseIdleConnections() {
	trr.ResolverSystem.CloseIdleConnections()
}

// LookupHost implements model.Resolver.LookupHost
func (trr *TrustedRecursiveResolver2) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	reso := trr.NewParallelDNSOverHTTPSResolverFn(trr.Logger, trr.URL)
	if reso == nil {
		reso = netxlite.NewParallelDNSOverHTTPSResolver(trr.Logger, trr.URL)
	}
	addrs, err := reso.LookupHost(ctx, hostname)
	if err != nil {
		addrs, err = trr.ResolverSystem.LookupHost(ctx, hostname)
	}
	return addrs, err
}

// LookupHTTPS implements model.Resolver.LookupHTTPS
func (trr *TrustedRecursiveResolver2) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	reso := trr.NewParallelDNSOverHTTPSResolverFn(trr.Logger, trr.URL)
	if reso == nil {
		reso = netxlite.NewParallelDNSOverHTTPSResolver(trr.Logger, trr.URL)
	}
	out, err := reso.LookupHTTPS(ctx, domain)
	if err != nil {
		out, err = trr.ResolverSystem.LookupHTTPS(ctx, domain)
	}
	return out, err
}

// LookupNS implements model.Resolver.LookupNS
func (trr *TrustedRecursiveResolver2) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	reso := trr.NewParallelDNSOverHTTPSResolverFn(trr.Logger, trr.URL)
	if reso == nil {
		reso = netxlite.NewParallelDNSOverHTTPSResolver(trr.Logger, trr.URL)
	}
	out, err := reso.LookupNS(ctx, domain)
	if err != nil {
		out, err = trr.ResolverSystem.LookupNS(ctx, domain)
	}
	return out, err
}

// Url returns the configured DoH URL for the TRR2 resolver
func (trr *TrustedRecursiveResolver2) Url() string {
	return trr.URL
}
