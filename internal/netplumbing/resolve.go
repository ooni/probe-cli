package netplumbing

import (
	"context"
	"net"
)

// Resolver performs domain name resolutions.
type Resolver interface {
	// LookupHost maps a domain name to IP addresses. If domain is an IP
	// address, this function returns a list containing such IP address
	// as the unique list element, and no error (like getaddrinfo).
	LookupHost(ctx context.Context, domain string) (addrs []string, err error)
}

// ErrResolve is an error occurred when resolving a domain name.
type ErrResolve struct {
	error
}

// Unwrap returns the wrapped error.
func (err *ErrResolve) Unwrap() error {
	return err.error
}

// LookupHost implements Resolver.LookupHost.
func (txp *Transport) LookupHost(ctx context.Context, domain string) ([]string, error) {
	if net.ParseIP(domain) != nil {
		return []string{domain}, nil // behave like getaddrinfo
	}
	log := txp.logger(ctx)
	log.Debugf("lookupHost %s...", domain)
	addresses, err := txp.routeLookupHost(ctx, domain)
	if err != nil {
		log.Debugf("lookupHost %s... %s", domain, err)
		return nil, &ErrResolve{err}
	}
	log.Debugf("lookupHost %s... %v", domain, addresses)
	return addresses, nil
}

// DefaultResolver is the resolver used by Transport.DefaultLookupHost.
var DefaultResolver = &net.Resolver{}

// routeLookupHost routes LookupHost calls.
func (txp *Transport) routeLookupHost(ctx context.Context, domain string) ([]string, error) {
	if config := ContextConfig(ctx); config != nil && config.Resolver != nil {
		return config.Resolver.LookupHost(ctx, domain)
	}
	return DefaultResolver.LookupHost(ctx, domain)
}
