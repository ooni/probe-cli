package netxlite

//
// Legacy code
//

// These vars export internal names to legacy ooni/probe-cli code.
//
// Deprecated: do not use these names in new code.
var (
	DefaultDialer     = &DialerSystem{}
	NewResolverSystem = newResolverSystem
	DefaultResolver   = newResolverSystem()
)

// These types export internal names to legacy ooni/probe-cli code.
//
// Deprecated: do not use these names in new code.
type (
	HTTPTransportWrapper           = httpTransportConnectionsCloser
	HTTPTransportLogger            = httpTransportLogger
	ErrorWrapperResolver           = resolverErrWrapper
	ResolverSystemDoNotInstantiate = resolverSystem // instantiate => crash w/ nil transport
	ResolverLogger                 = resolverLogger
	ResolverIDNA                   = resolverIDNA
	AddressResolver                = resolverShortCircuitIPAddr
)
