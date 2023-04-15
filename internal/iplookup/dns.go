package iplookup

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// newFamilyResolver creates a new [model.Resolver] using the given family
// and the underlying [model.Resolver] used by the [Client].
func (c *Client) newFamilyResolver(family Family) *familyResolver {
	return &familyResolver{
		family: family,
		reso:   c.Resolver,
	}
}

// familyResolver is a resolver that only considers a given address family.
type familyResolver struct {
	family Family
	reso   model.Resolver
}

var _ model.Resolver = &familyResolver{}

// Address implements model.Resolver
func (fr *familyResolver) Address() string {
	return fr.reso.Address()
}

// CloseIdleConnections implements model.Resolver
func (fr *familyResolver) CloseIdleConnections() {
	fr.reso.CloseIdleConnections()
}

// LookupHTTPS implements model.Resolver
func (fr *familyResolver) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return nil, netxlite.ErrNoDNSTransport
}

// LookupHost implements model.Resolver
func (fr *familyResolver) LookupHost(ctx context.Context, domain string) (addrs []string, err error) {
	// make sure the DNS lookup does not block us forever
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	// resolve the domain name to IP addresses using the child resolver
	all, err := fr.reso.LookupHost(ctx, domain)
	if err != nil {
		return nil, err
	}

	// filter the addresses we want
	var filtered []string
	for _, addr := range all {
		ipv6, err := netxlite.IsIPv6(addr)
		if err != nil {
			// should not happen
			continue
		}
		switch {
		case fr.family == FamilyINET && !ipv6:
			filtered = append(filtered, addr)
		case fr.family == FamilyINET6 && ipv6:
			filtered = append(filtered, addr)
		}
	}

	// handle the case where there's no available address
	if len(filtered) < 1 {
		return nil, netxlite.NewTopLevelGenericErrWrapper(netxlite.ErrOODNSNoAnswer)
	}

	return filtered, nil

}

// LookupNS implements model.Resolver
func (fr *familyResolver) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	return nil, netxlite.ErrNoDNSTransport
}

// Network implements model.Resolver
func (fr *familyResolver) Network() string {
	return fr.reso.Network()
}
