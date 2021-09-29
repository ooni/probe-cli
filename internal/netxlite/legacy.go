package netxlite

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/lucas-clemente/quic-go"
)

// These vars export internal names to legacy ooni/probe-cli code.
//
// Deprecated: do not use these names in new code.
var (
	DefaultDialer        = &dialerSystem{}
	DefaultTLSHandshaker = defaultTLSHandshaker
	NewConnUTLS          = newConnUTLS
	DefaultResolver      = &resolverSystem{}
)

// These types export internal names to legacy ooni/probe-cli code.
//
// Deprecated: do not use these names in new code.
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
	AddressResolver           = resolverShortCircuitIPAddr
)

// ResolverLegacy performs domain name resolutions.
//
// Depecated: new code should use Resolver.
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

// ResolverLegacyAdapter makes a ResolverLegacy behave like a Resolver.
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

// LookupHTTPS always returns ErrDNSNoTransport.
func (r *ResolverLegacyAdapter) LookupHTTPS(
	ctx context.Context, domain string) (*HTTPSSvc, error) {
	return nil, ErrNoDNSTransport
}

// DialerLegacy establishes network connections.
//
// Deprecated: please use Dialer instead.
//
// Existing code in probe-cli can use it until we
// have finished refactoring it.
type DialerLegacy interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// NewDialerLegacyAdapter adapts a DialerrLegacy to
// become compatible with the Dialer definition.
//
// Deprecated: do not use this function in new code.
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

// CloseIdleConnections implements Dialer.CloseIdleConnections.
func (d *DialerLegacyAdapter) CloseIdleConnections() {
	if ra, ok := d.DialerLegacy.(dialerLegacyIdleConnectionsCloser); ok {
		ra.CloseIdleConnections()
	}
}

// QUICContextDialer is a dialer for QUIC using Context.
//
// Deprecated: new code should use QUICDialer.
//
// Use NewQUICDialerFromContextDialerAdapter if you need to
// adapt to QUICDialer.
type QUICContextDialer interface {
	// DialContext establishes a new QUIC session using the given
	// network and address. The tlsConfig and the quicConfig arguments
	// MUST NOT be nil. Returns either the session or an error.
	DialContext(ctx context.Context, network, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error)
}

// NewQUICDialerFromContextDialerAdapter creates a new
// QUICDialer from a QUICContextDialer.
func NewQUICDialerFromContextDialerAdapter(d QUICContextDialer) QUICDialer {
	return &QUICContextDialerAdapter{d}
}

// QUICContextDialerAdapter adapts a QUICContextDialer to be a QUICDialer.
type QUICContextDialerAdapter struct {
	QUICContextDialer
}

type quicContextDialerConnectionsCloser interface {
	CloseIdleConnections()
}

// CloseIdleConnections implements QUICDialer.CloseIdleConnections.
func (d *QUICContextDialerAdapter) CloseIdleConnections() {
	if o, ok := d.QUICContextDialer.(quicContextDialerConnectionsCloser); ok {
		o.CloseIdleConnections()
	}
}
