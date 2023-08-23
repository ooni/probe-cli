package netxlite

//
// Net is a high-level structure that provides constructors for basic netxlite network operations
// using a custom Underlying Network.
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

// Net contains a [model.UnderlyingNetwork] to perform network operations.
type Net struct {
	Underlying model.UnderlyingNetwork
}

func (n *Net) tproxyNilSafeProvider() *tproxyNilSafeProvider {
	return &tproxyNilSafeProvider{n.Underlying}
}

// NewStdlibResolver is like netxlite.NewStdlibResolver but
// the constructed resolver uses the given [UnderlyingNetwork].
func (n *Net) NewStdlibResolver(logger model.DebugLogger, wrappers ...model.DNSTransportWrapper) model.Resolver {
	unwrapped := &resolverSystem{
		t: WrapDNSTransport(&dnsOverGetaddrinfoTransport{provider: n.tproxyNilSafeProvider()}, wrappers...),
	}
	return WrapResolver(logger, unwrapped)
}

// NewDialerWithResolver is like netxlite.NewDialerWithResolver but
// the constructed dialer uses the given [UnderlyingNetwork].
func (n *Net) NewDialerWithResolver(dl model.DebugLogger, r model.Resolver, w ...model.DialerWrapper) model.Dialer {
	return WrapDialer(dl, r, &DialerSystem{provider: n.tproxyNilSafeProvider()}, w...)
}

// NewQUICListener is like netxlite.NewQUICListener but
// the constructed listener uses the given [UnderlyingNetwork].
func (n *Net) NewQUICListener() model.QUICListener {
	return &quicListenerErrWrapper{&quicListenerStdlib{provider: n.tproxyNilSafeProvider()}}
}

// NewQUICDialerWithResolver is like netxlite.NewQUICDialerWithResolver but
// the constructed QUIC dialer uses the given [UnderlyingNetwork].
func (n *Net) NewQUICDialerWithResolver(listener model.QUICListener, logger model.DebugLogger,
	resolver model.Resolver, wrappers ...model.QUICDialerWrapper) (outDialer model.QUICDialer) {
	baseDialer := &quicDialerQUICGo{
		QUICListener: listener,
		provider:     n.tproxyNilSafeProvider(),
	}
	return WrapQUICDialer(logger, resolver, baseDialer, wrappers...)
}

// NewTLSHandshakerStdlib is like netxlite.NewTLSHandshakerStdlib but
// the constructed handshaker uses the given [UnderlyingNetwork].
func (n *Net) NewTLSHandshakerStdlib(logger model.DebugLogger) model.TLSHandshaker {
	return newTLSHandshakerLogger(&tlsHandshakerConfigurable{provider: n.tproxyNilSafeProvider()}, logger)
}

// NewHTTPTransportStdlib is like netxlite.NewHTTPTransportStdlib but
// the constructed transport uses the given [UnderlyingNetwork].
func (n *Net) NewHTTPTransportStdlib(logger model.DebugLogger) model.HTTPTransport {
	dialer := n.NewDialerWithResolver(logger, n.NewStdlibResolver(logger))
	tlsDialer := NewTLSDialer(dialer, n.NewTLSHandshakerStdlib(logger))
	return NewHTTPTransport(logger, dialer, tlsDialer)
}

// NewHTTP3TransportStdlib is like netxlite.NewHTTP3TransportStdlib but
// the constructed transport uses the given [UnderlyingNetwork].
func (n *Net) NewHTTP3TransportStdlib(logger model.DebugLogger) model.HTTPTransport {
	ql := n.NewQUICListener()
	reso := n.NewStdlibResolver(logger)
	qd := n.NewQUICDialerWithResolver(ql, logger, reso)
	return NewHTTP3Transport(logger, qd, nil)
}
