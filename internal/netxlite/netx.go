package netxlite

//
// Netx is a high-level structure that provides constructors for basic netxlite
// network operations using a custom model.UnderlyingNetwork.
//

import "github.com/ooni/probe-cli/v3/internal/model"

// TODO(bassosimone,kelmenhorst): we should gradually refactor the top-level netxlite
// functions to operate on a [Netx] struct using a nil-initialized Underlying field.

// Netx allows constructing netxlite data types using a specific [model.UnderlyingNetwork].
type Netx struct {
	// Underlying is the OPTIONAL [model.UnderlyingNetwork] to use. Leaving this field
	// nil makes this implementation functionally equivalent to netxlite top-level functions.
	Underlying model.UnderlyingNetwork
}

// tproxyNilSafeProvider wraps the [model.UnderlyingNetwork] using a [tproxyNilSafeProvider].
func (netx *Netx) tproxyNilSafeProvider() *MaybeCustomUnderlyingNetwork {
	return &MaybeCustomUnderlyingNetwork{netx.Underlying}
}

// NewDialerWithResolver is like [netxlite.NewDialerWithResolver] but the constructed [model.Dialer]
// uses the [model.UnderlyingNetwork] configured inside the [Netx] structure.
func (n *Netx) NewDialerWithResolver(dl model.DebugLogger, r model.Resolver, w ...model.DialerWrapper) model.Dialer {
	return WrapDialer(dl, r, &DialerSystem{provider: n.tproxyNilSafeProvider()}, w...)
}

// NewUDPListener is like [netxlite.NewUDPListener] but the constructed [model.UDPListener]
// uses the [model.UnderlyingNetwork] configured inside the [Netx] structure.
func (n *Netx) NewUDPListener() model.UDPListener {
	return &udpListenerErrWrapper{&udpListenerStdlib{provider: n.tproxyNilSafeProvider()}}
}

// NewQUICDialerWithResolver is like [netxlite.NewQUICDialerWithResolver] but the constructed
// [model.QUICDialer] uses the [model.UnderlyingNetwork] configured inside the [Netx] structure.
func (n *Netx) NewQUICDialerWithResolver(listener model.UDPListener, logger model.DebugLogger,
	resolver model.Resolver, wrappers ...model.QUICDialerWrapper) (outDialer model.QUICDialer) {
	baseDialer := &quicDialerQUICGo{
		UDPListener: listener,
		provider:    n.tproxyNilSafeProvider(),
	}
	return WrapQUICDialer(logger, resolver, baseDialer, wrappers...)
}

// NewTLSHandshakerStdlib is like [netxlite.NewTLSHandshakerStdlib] but the constructed [model.TLSHandshaker]
// uses the [model.UnderlyingNetwork] configured inside the [Netx] structure.
func (n *Netx) NewTLSHandshakerStdlib(logger model.DebugLogger) model.TLSHandshaker {
	return newTLSHandshakerLogger(&tlsHandshakerConfigurable{provider: n.tproxyNilSafeProvider()}, logger)
}

// NewHTTPTransportStdlib is like [netxlite.NewHTTPTransportStdlib] but the constructed [model.HTTPTransport]
// uses the [model.UnderlyingNetwork] configured inside the [Netx] structure.
func (n *Netx) NewHTTPTransportStdlib(logger model.DebugLogger) model.HTTPTransport {
	dialer := n.NewDialerWithResolver(logger, n.NewStdlibResolver(logger))
	tlsDialer := NewTLSDialer(dialer, n.NewTLSHandshakerStdlib(logger))
	return NewHTTPTransport(logger, dialer, tlsDialer)
}

// NewHTTP3TransportStdlib is like [netxlite.NewHTTP3TransportStdlib] but the constructed [model.HTTPTransport]
// uses the [model.UnderlyingNetwork] configured inside the [Netx] structure.
func (n *Netx) NewHTTP3TransportStdlib(logger model.DebugLogger) model.HTTPTransport {
	ql := n.NewUDPListener()
	reso := n.NewStdlibResolver(logger)
	qd := n.NewQUICDialerWithResolver(ql, logger, reso)
	return NewHTTP3Transport(logger, qd, nil)
}
