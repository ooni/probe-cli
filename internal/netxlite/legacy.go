package netxlite

//
// Legacy code
//

// These vars export internal names to legacy ooni/probe-cli code.
//
// Deprecated: do not use these names in new code.
var (
	DefaultDialer        = &DialerSystem{}
	DefaultTLSHandshaker = defaultTLSHandshaker
	NewResolverSystem    = newResolverSystem
	NewConnUTLS          = newConnUTLS
	DefaultResolver      = newResolverSystem()
)

// These types export internal names to legacy ooni/probe-cli code.
//
// Deprecated: do not use these names in new code.
type (
	DialerResolver                 = dialerResolver
	DialerLogger                   = dialerLogger
	HTTPTransportWrapper           = httpTransportConnectionsCloser
	HTTPTransportLogger            = httpTransportLogger
	ErrorWrapperDialer             = dialerErrWrapper
	ErrorWrapperQUICListener       = quicListenerErrWrapper
	ErrorWrapperQUICDialer         = quicDialerErrWrapper
	ErrorWrapperResolver           = resolverErrWrapper
	ErrorWrapperTLSHandshaker      = tlsHandshakerErrWrapper
	QUICListenerStdlib             = quicListenerStdlib
	QUICDialerQUICGo               = quicDialerQUICGo
	QUICDialerResolver             = quicDialerResolver
	QUICDialerLogger               = quicDialerLogger
	ResolverSystemDoNotInstantiate = resolverSystem // instantiate => crash w/ nil transport
	ResolverLogger                 = resolverLogger
	ResolverIDNA                   = resolverIDNA
	TLSHandshakerConfigurable      = tlsHandshakerConfigurable
	TLSHandshakerLogger            = tlsHandshakerLogger
	TLSDialerLegacy                = tlsDialer
	AddressResolver                = resolverShortCircuitIPAddr
)
