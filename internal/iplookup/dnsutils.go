package iplookup

//
// Common DNS code
//

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// familyAwareLookupHostWithImplicitTimeout is a DNS lookup host operation that
// is aware of the address family and has an implicit timeout.
func (c *Client) familyAwareLookupHostWithImplicitTimeout(
	ctx context.Context, family model.AddressFamily, domain string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	r := c.newAddressFamilyResolver(family)
	return r.LookupHost(ctx, domain)
}

// newAddressFamilyResolver creates a new [model.Resolver] using the given address
// family and the underlying [model.Resolver] used by the [Client].
func (c *Client) newAddressFamilyResolver(family model.AddressFamily) model.Resolver {
	return netxlite.NewAddressFamilyResolver(c.resolver, family)
}
