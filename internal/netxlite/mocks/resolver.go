package mocks

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx/model"
)

// Resolver is a mockable Resolver.
type Resolver struct {
	MockLookupHost           func(ctx context.Context, domain string) ([]string, error)
	MockNetwork              func() string
	MockAddress              func() string
	MockCloseIdleConnections func()
}

// LookupHost calls MockLookupHost.
func (r *Resolver) LookupHost(ctx context.Context, domain string) ([]string, error) {
	return r.MockLookupHost(ctx, domain)
}

// Address calls MockAddress.
func (r *Resolver) Address() string {
	return r.MockAddress()
}

// Network calls MockNetwork.
func (r *Resolver) Network() string {
	return r.MockNetwork()
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (r *Resolver) CloseIdleConnections() {
	r.MockCloseIdleConnections()
}

func (r *Resolver) LookupHostWithoutRetry(
	ctx context.Context, domain string, qtype uint16) ([]string, error) {
	panic("not yet implemented")
}

// HTTPS is an HTTPS reply.
type HTTPS = model.HTTPS

func (r *Resolver) LookupHTTPSWithoutRetry(
	ctx context.Context, domain string) (HTTPS, error) {
	panic("not yet implemented")
}
