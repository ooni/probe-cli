package oonet

import (
	"context"
	"net"
)

// DefaultResolver is the resolver used by Transport.DefaultLookupHost.
var DefaultResolver = &net.Resolver{}

// ErrResolve is an error occurred when resolving a domain name.
type ErrResolve struct {
	error
}

// Unwrap returns the wrapped error.
func (err *ErrResolve) Unwrap() error {
	return err.error
}

// LookupHost maps domain to a list of IP addresses. On error, this function will always
// return an ErrResolve instance. If txp.Logger is not nil, this function will use it. If
// ContextOverrides().LookupHost is configured, this function uses it; otherwise, it defaults
// to Transport.DefaultLookupHost, which uses DefaultResolver. If domain is an IP address,
// this function will immediately return a list containing it without performing any DNS
// network operation (this is what getaddrinfo also does).
func (txp *Transport) LookupHost(ctx context.Context, domain string) ([]string, error) {
	if net.ParseIP(domain) != nil {
		return []string{domain}, nil // behave like getaddrinfo
	}
	log := txp.logger()
	log.Debugf("lookupHost %s...", domain)
	addresses, err := txp.routeLookupHost(ctx, domain)
	if err != nil {
		log.Debugf("lookupHost %s... %s", domain, err)
		return nil, &ErrResolve{err}
	}
	log.Debugf("lookupHost %s... %v", domain, addresses)
	return addresses, nil
}

// routeLookupHost routes LookupHost calls.
func (txp *Transport) routeLookupHost(ctx context.Context, domain string) ([]string, error) {
	if overrides := ContextOverrides(ctx); overrides != nil && overrides.LookupHost != nil {
		return overrides.LookupHost(ctx, domain)
	}
	return DefaultResolver.LookupHost(ctx, domain)
}
