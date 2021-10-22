package resolver

import (
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// IDNAResolver is to support resolving Internationalized Domain Names.
// See RFC3492 for more information.
type IDNAResolver = netxlite.ResolverIDNA

var _ Resolver = &IDNAResolver{}
