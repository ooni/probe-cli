package resolver

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// BogonResolver is a bogon aware resolver. When a bogon is encountered in
// a reply, this resolver will return an error.
//
// Deprecation warning
//
// This resolver is deprecated. The right thing to do would be to check
// for bogons right after a domain name resolution in the nettest.
type BogonResolver struct {
	model.Resolver
}

// LookupHost implements Resolver.LookupHost
func (r BogonResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	for _, addr := range addrs {
		if netxlite.IsBogon(addr) {
			return nil, netxlite.ErrDNSBogon
		}
	}
	return addrs, err
}

var _ model.Resolver = BogonResolver{}
