package mocks

import "context"

// Resolver is a mockable Resolver.
type Resolver struct {
	MockLookupHost func(ctx context.Context, domain string) ([]string, error)
	MockNetwork    func() string
	MockAddress    func() string
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
