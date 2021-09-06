package netxlite

import (
	"context"
	"net"
)

// These vars export internal names to legacy ooni/probe-cli code.
var (
	DefaultDialer        = defaultDialer
	DefaultTLSHandshaker = defaultTLSHandshaker
	NewConnUTLS          = newConnUTLS
)

// These types export internal names to legacy ooni/probe-cli code.
type (
	DialerResolver            = dialerResolver
	DialerLogger              = dialerLogger
	HTTPTransportLogger       = httpTransportLogger
	QUICListenerStdlib        = quicListenerStdlib
	QUICDialerQUICGo          = quicDialerQUICGo
	QUICDialerResolver        = quicDialerResolver
	QUICDialerLogger          = quicDialerLogger
	ResolverSystem            = resolverSystem
	ResolverLogger            = resolverLogger
	ResolverIDNA              = resolverIDNA
	TLSHandshakerConfigurable = tlsHandshakerConfigurable
	TLSHandshakerLogger       = tlsHandshakerLogger
	DialerSystem              = dialerSystem
	TLSDialerLegacy           = tlsDialer
)

// ResolverLegacy performs domain name resolutions.
//
// This definition of Resolver is DEPRECATED. New code should use
// the more complete definition in the new Resolver interface.
//
// Existing code in ooni/probe-cli is still using this definition.
type ResolverLegacy interface {
	// LookupHost behaves like net.Resolver.LookupHost.
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)
}

// NewResolverLegacyAdapter adapts a ResolverLegacy to
// become compatible with the Resolver definition.
func NewResolverLegacyAdapter(reso ResolverLegacy) Resolver {
	return &ResolverLegacyAdapter{reso}
}

// ResolverLegacyAdapter makes a ResolverLegacy behave like
// it was a Resolver type. If ResolverLegacy is actually also
// a Resolver, this adapter will just forward missing calls,
// otherwise it will implement a sensible default action.
type ResolverLegacyAdapter struct {
	ResolverLegacy
}

var _ Resolver = &ResolverLegacyAdapter{}

type resolverLegacyNetworker interface {
	Network() string
}

// Network implements Resolver.Network.
func (r *ResolverLegacyAdapter) Network() string {
	if rn, ok := r.ResolverLegacy.(resolverLegacyNetworker); ok {
		return rn.Network()
	}
	return "adapter"
}

type resolverLegacyAddresser interface {
	Address() string
}

// Address implements Resolver.Address.
func (r *ResolverLegacyAdapter) Address() string {
	if ra, ok := r.ResolverLegacy.(resolverLegacyAddresser); ok {
		return ra.Address()
	}
	return ""
}

type resolverLegacyIdleConnectionsCloser interface {
	CloseIdleConnections()
}

// CloseIdleConnections implements Resolver.CloseIdleConnections.
func (r *ResolverLegacyAdapter) CloseIdleConnections() {
	if ra, ok := r.ResolverLegacy.(resolverLegacyIdleConnectionsCloser); ok {
		ra.CloseIdleConnections()
	}
}

// DialerLegacy establishes network connections.
//
// This definition is DEPRECATED. Please, use Dialer.
//
// Existing code in probe-cli can use it until we
// have finished refactoring it.
type DialerLegacy interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// NewDialerLegacyAdapter adapts a DialerrLegacy to
// become compatible with the Dialer definition.
func NewDialerLegacyAdapter(d DialerLegacy) Dialer {
	return &DialerLegacyAdapter{d}
}

// DialerLegacyAdapter makes a DialerLegacy behave like
// it was a Dialer type. If DialerLegacy is actually also
// a Dialer, this adapter will just forward missing calls,
// otherwise it will implement a sensible default action.
type DialerLegacyAdapter struct {
	DialerLegacy
}

var _ Dialer = &DialerLegacyAdapter{}

type dialerLegacyIdleConnectionsCloser interface {
	CloseIdleConnections()
}

// CloseIdleConnections implements Resolver.CloseIdleConnections.
func (d *DialerLegacyAdapter) CloseIdleConnections() {
	if ra, ok := d.DialerLegacy.(dialerLegacyIdleConnectionsCloser); ok {
		ra.CloseIdleConnections()
	}
}
