package netxlite

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/errorsx"
)

// reduceErrors finds a known error in a list of errors since
// it's probably most relevant.
//
// Deprecation warning
//
// Albeit still used, this function is now DEPRECATED.
//
// In perspective, we would like to transition to a scenario where
// full dialing is NOT used for measurements and we return a multierror here.
func reduceErrors(errorslist []error) error {
	if len(errorslist) == 0 {
		return nil
	}
	// If we have a known error, let's consider this the real error
	// since it's probably most relevant. Otherwise let's return the
	// first considering that (1) local resolvers likely will give
	// us IPv4 first and (2) also our resolver does that. So, in case
	// the user has no IPv6 connectivity, an IPv6 error is going to
	// appear later in the list of errors.
	for _, err := range errorslist {
		var wrapper *errorsx.ErrWrapper
		if errors.As(err, &wrapper) && !strings.HasPrefix(
			err.Error(), "unknown_failure",
		) {
			return err
		}
	}
	// TODO(bassosimone): handle this case in a better way
	return errorslist[0]
}

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
