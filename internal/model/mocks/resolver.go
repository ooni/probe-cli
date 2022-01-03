package mocks

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Resolver is a mockable Resolver.
type Resolver struct {
	MockLookupHost           func(ctx context.Context, domain string) ([]string, error)
	MockNetwork              func() string
	MockAddress              func() string
	MockCloseIdleConnections func()
	MockLookupHTTPS          func(ctx context.Context, domain string) (*model.HTTPSSvc, error)
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

// LookupHTTPS calls MockLookupHTTPS.
func (r *Resolver) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return r.MockLookupHTTPS(ctx, domain)
}
