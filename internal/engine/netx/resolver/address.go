package resolver

import (
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// AddressResolver is a resolver that knows how to correctly
// resolve IP addresses to themselves.
type AddressResolver = netxlite.AddressResolver
