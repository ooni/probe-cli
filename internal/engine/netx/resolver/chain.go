package resolver

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// ChainResolver is a chain resolver. The primary resolver is used first and, if that
// fails, we then attempt with the secondary resolver.
type ChainResolver struct {
	Primary   model.Resolver
	Secondary model.Resolver
}

// LookupHost implements Resolver.LookupHost
func (c ChainResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := c.Primary.LookupHost(ctx, hostname)
	if err != nil {
		addrs, err = c.Secondary.LookupHost(ctx, hostname)
	}
	return addrs, err
}

// Network implements Resolver.Network
func (c ChainResolver) Network() string {
	return "chain"
}

// Address implements Resolver.Address
func (c ChainResolver) Address() string {
	return ""
}

// CloseIdleConnections implements Resolver.CloseIdleConnections.
func (c ChainResolver) CloseIdleConnections() {
	c.Primary.CloseIdleConnections()
	c.Secondary.CloseIdleConnections()
}

// LookupHTTPS implements Resolver.LookupHTTPS
func (c ChainResolver) LookupHTTPS(
	ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	https, err := c.Primary.LookupHTTPS(ctx, domain)
	if err != nil {
		https, err = c.Secondary.LookupHTTPS(ctx, domain)
	}
	return https, err
}

var _ model.Resolver = ChainResolver{}
