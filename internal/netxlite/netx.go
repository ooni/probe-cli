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

// maybeCustomUnderlyingNetwork wraps the [model.UnderlyingNetwork] using a [*MaybeCustomUnderlyingNetwork].
func (netx *Netx) maybeCustomUnderlyingNetwork() *MaybeCustomUnderlyingNetwork {
	return &MaybeCustomUnderlyingNetwork{netx.Underlying}
}

// NewTLSHandshakerStdlib is like [netxlite.NewTLSHandshakerStdlib] but the constructed [model.TLSHandshaker]
// uses the [model.UnderlyingNetwork] configured inside the [Netx] structure.
func (n *Netx) NewTLSHandshakerStdlib(logger model.DebugLogger) model.TLSHandshaker {
	return newTLSHandshakerLogger(&tlsHandshakerConfigurable{provider: n.maybeCustomUnderlyingNetwork()}, logger)
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
