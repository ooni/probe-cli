// Package resolverlookup contains code to discover the addresses
// used by system resolver (i.e., getaddrinfo).
package resolverlookup

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Client is the resolver lookup client. The zero value of this struct
// is invalid; please fill the fields marked as MANDATORY.
type Client struct {
	// Logger is the MANDATORY logger to use.
	Logger model.Logger
}

// LookupResolverIPv4 returns the IPv4 address used by the system resolver.
func (c *Client) LookupResolverIPv4(ctx context.Context) (string, error) {
	// MUST be the system resolver! See https://github.com/ooni/probe/issues/2360
	reso := netxlite.NewStdlibResolver(c.Logger)
	var ips []string
	ips, err := reso.LookupHost(ctx, "whoami.v4.powerdns.org")
	if err != nil {
		return "", err
	}
	// Note: it feels okay to panic here because a resolver is expected to never return
	// zero valid IP addresses to the caller without emitting an error.
	runtimex.Assert(len(ips) >= 1, "reso.LookupHost returned zero IP addresses")
	return ips[0], nil
}

// LookupResolverIPv6 returns the IPv6 address used by the system resolver.
func (c *Client) LookupResolverIPv6(ctx context.Context) (string, error) {
	// MUST be the system resolver! See https://github.com/ooni/probe/issues/2360
	reso := netxlite.NewStdlibResolver(c.Logger)
	var ips []string
	ips, err := reso.LookupHost(ctx, "whoami.v6.powerdns.org")
	if err != nil {
		return "", err
	}
	// Note: it feels okay to panic here because a resolver is expected to never return
	// zero valid IP addresses to the caller without emitting an error.
	runtimex.Assert(len(ips) >= 1, "reso.LookupHost returned zero IP addresses")
	return ips[0], nil
}
