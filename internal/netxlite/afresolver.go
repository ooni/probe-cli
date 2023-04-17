package netxlite

//
// Address-family resolver - limiting the results of a DNS
// lookup operation to a specific address family only.
//

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// NewAddressFamilyResolver creates a new [model.Resolver] using the given
// [AddressFamily] and the underlying [model.Resolver].
//
// For example, if you provide [AddressFamilyINET6] as the family argument, this
// resolver will only return IPv6 addresses for each lookup regardless of whether
// the actual result of the lookup contained IPv4 addresses as well.
//
// This function panics if the family argument is not one of:
//
// - [AddressFamilyINET]
//
// - [AddressFamilyINET6]
//
// The returned resolver will only filter the results of LookupHost since the
// result of other lookup functions is never ambiguous.
func NewAddressFamilyResolver(reso model.Resolver, family model.AddressFamily) model.Resolver {
	runtimex.Assert(
		family == model.AddressFamilyINET || family == model.AddressFamilyINET6,
		"NewAddressFamilyResolver: invalid family argument",
	)
	return &addressFamilyResolver{
		family: family,
		reso:   reso,
	}
}

// addressFamilyResolver is the [model.Resolver] returned by
// the [NewAddressFamilyResolver] constructor.
type addressFamilyResolver struct {
	// family is the family to which we're exclusively interested.
	family model.AddressFamily

	// reso is the underlying resolver.
	reso model.Resolver
}

var _ model.Resolver = &addressFamilyResolver{}

// Address implements model.Resolver
func (afr *addressFamilyResolver) Address() string {
	return afr.reso.Address()
}

// CloseIdleConnections implements model.Resolver
func (afr *addressFamilyResolver) CloseIdleConnections() {
	afr.reso.CloseIdleConnections()
}

// LookupHTTPS implements model.Resolver
func (afr *addressFamilyResolver) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	// Implementation note: passthrough is fine here since addresses are
	// already divided by their protocol family
	return afr.reso.LookupHTTPS(ctx, domain)
}

// LookupHost implements model.Resolver
func (afr *addressFamilyResolver) LookupHost(ctx context.Context, domain string) (addrs []string, err error) {
	// resolve the domain name to IP addresses using the child resolver
	all, err := afr.reso.LookupHost(ctx, domain)
	if err != nil {
		return nil, err
	}

	// filter the addresses we want
	var filtered []string
	for _, addr := range all {
		ipv6, err := IsIPv6(addr)
		if err != nil {
			// should not happen
			continue
		}
		switch {
		case afr.family == model.AddressFamilyINET && !ipv6:
			filtered = append(filtered, addr)
		case afr.family == model.AddressFamilyINET6 && ipv6:
			filtered = append(filtered, addr)
		}
	}

	// handle the case where there's no available address
	if len(filtered) < 1 {
		return nil, NewTopLevelGenericErrWrapper(ErrOODNSNoAnswer)
	}

	return filtered, nil

}

// LookupNS implements model.Resolver
func (afr *addressFamilyResolver) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	// Implementation note: passthrough is fine here since the return
	// value consists of strings containing domain names.
	return afr.reso.LookupNS(ctx, domain)
}

// Network implements model.Resolver
func (afr *addressFamilyResolver) Network() string {
	return afr.reso.Network()
}
