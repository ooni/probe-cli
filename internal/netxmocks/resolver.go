package netxmocks

import "context"

// resolver is the interface we expect from a resolver
type resolver interface {
	LookupHost(ctx context.Context, domain string) ([]string, error)
	Network() string
	Address() string
}

// Resolver is a mockable Resolver.
type Resolver struct {
	MockLookupHost func(ctx context.Context, domain string) ([]string, error)
	MockNetwork    func() string
	MockAddress    func() string
}

// LookupHost implements Resolver.LookupHost.
func (r *Resolver) LookupHost(ctx context.Context, domain string) ([]string, error) {
	return r.MockLookupHost(ctx, domain)
}

// Address implements Resolver.Address.
func (r *Resolver) Address() string {
	return r.MockAddress()
}

// Network implements Resolver.Network.
func (r *Resolver) Network() string {
	return r.MockNetwork()
}

var _ resolver = &Resolver{}
