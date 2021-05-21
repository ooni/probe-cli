package netplumbing

// This file contains the implementation of Transport.LookupHost.

import (
	"context"
	"net"
	"time"
)

// LookupHost resolves a domain name to a list of IP addresses.
func (txp *Transport) LookupHost(ctx context.Context, domain string) ([]string, error) {
	if net.ParseIP(domain) != nil {
		return []string{domain}, nil // behave like getaddrinfo
	}
	log := txp.logger(ctx)
	log.Debugf("lookupHost: %s...", domain)
	addresses, err := txp.lookupHostMaybeTrace(ctx, domain)
	if err != nil {
		log.Debugf("lookupHost: %s... %s", domain, err)
		return nil, &ErrResolve{err}
	}
	log.Debugf("lookupHost: %s... %v", domain, addresses)
	return addresses, nil
}

// ErrResolve is an error occurred when resolving a domain name.
type ErrResolve struct {
	error
}

// Unwrap returns the wrapped error.
func (err *ErrResolve) Unwrap() error {
	return err.error
}

// lookupHostMaybeTrace enables tracing if needed.
func (txp *Transport) lookupHostMaybeTrace(
	ctx context.Context, domain string) ([]string, error) {
	if th := ContextTraceHeader(ctx); th != nil {
		return txp.lookupHostWithTraceHeader(ctx, domain, th)
	}
	return txp.lookupHostMaybeOverride(ctx, domain)
}

// lookupHostWithTraceHeader traces a lookupHost.
func (txp *Transport) lookupHostWithTraceHeader(
	ctx context.Context, domain string, th *TraceHeader) ([]string, error) {
	ev := &LookupHostTrace{
		Domain:    domain,
		StartTime: time.Now(),
	}
	child := &TraceHeader{}
	addrs, err := txp.lookupHostMaybeOverride(WithTraceHeader(ctx, child), domain)
	ev.EndTime = time.Now()
	ev.Addresses = addrs
	ev.Error = err
	ev.Children = child.MoveOut()
	th.add(ev)
	return addrs, err
}

// LookupHostTrace is a measurement performed during a lookup host.
type LookupHostTrace struct {
	// Domain is the domain to resolve.
	Domain string

	// StartTime is when we started the resolve.
	StartTime time.Time

	// EndTime is when we're done.
	EndTime time.Time

	// Addresses contains the resolver addresses.
	Addresses []string

	// Error contains the error.
	Error error

	// Children contains the children events.
	Children []TraceEvent
}

// Kind implements TraceEvent.Kind.
func (te *LookupHostTrace) Kind() string {
	return TraceKindLookupHost
}

// lookupHostMaybeOverride uses the overriden or the custom resolver.
func (txp *Transport) lookupHostMaybeOverride(
	ctx context.Context, domain string) ([]string, error) {
	reso := txp.DefaultResolver()
	// avoid looping when a lookupHost needs another to bootstrap
	if !contextLookupHostLoopFlag(ctx) {
		if config := ContextConfig(ctx); config != nil && config.Resolver != nil {
			reso = config.Resolver
		}
		ctx = withLookupHostLoopFlag(ctx)
	}
	return reso.LookupHost(ctx, domain)
}

// lookupHostLoopFlagKey is a key indicating that we are inside
// a lookupHost operation that has called lookupHost. This
// happens, for example, when using `dot://dns.google`. When
// this happens, we fallback to the default resolver, thus
// we do not loop forever calling ourselves.
type lookupHostLoopFlagKey struct{}

// withLookupHostLoopFlag sets the lookupHost loop flag.
func withLookupHostLoopFlag(ctx context.Context) context.Context {
	return context.WithValue(ctx, lookupHostLoopFlagKey{}, true)
}

// contextLookupHostLoopFlag returns the lookupHost loop flag.
func contextLookupHostLoopFlag(ctx context.Context) bool {
	v, _ := ctx.Value(lookupHostLoopFlagKey{}).(bool)
	return v
}

// DefaultResolver returns the default Resolver used by this Transport.
func (txp *Transport) DefaultResolver() Resolver {
	return &net.Resolver{}
}
