package netxlite

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
	ErrorWrapperDialer        = dialerErrWrapper
	ErrorWrapperQUICListener  = quicListenerErrWrapper
	ErrorWrapperQUICDialer    = quicDialerErrWrapper
	ErrorWrapperResolver      = resolverErrWrapper
	ErrorWrapperTLSHandshaker = tlsHandshakerErrWrapper
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
