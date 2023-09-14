package netxlite

//
// Legacy HTTP code (mainly used by ./internal/legacy/netx)
//

import (
	"net/http"

	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// NewHTTPTransportWithResolverLegacy creates a new HTTP transport using
// the stdlib for everything but the given resolver.
//
// Deprecated: do not use this func in new code.
func NewHTTPTransportWithResolverLegacy(logger model.DebugLogger, reso model.Resolver) model.HTTPTransport {
	dialer := NewDialerWithResolver(logger, reso)
	thx := NewTLSHandshakerStdlib(logger)
	tlsDialer := NewTLSDialer(dialer, thx)
	return NewHTTPTransportLegacy(logger, dialer, tlsDialer)
}

// NewHTTPTransportLegacy returns a wrapped HTTP transport for HTTP2 and HTTP/1.1
// using the given dialer and logger.
//
// Deprecated: do not use this func in new code.
//
// The returned transport will gracefully handle TLS connections
// created using gitlab.com/yawning/utls.git, if the TLS dialer
// is a dialer using such library for TLS operations.
//
// The returned transport will not have a configured proxy, not
// even the proxy configurable from the environment.
//
// QUIRK: the returned transport will disable transparent decompression
// of compressed response bodies (and will not automatically
// ask for such compression, though you can always do that manually).
//
// The returned transport will configure TCP and TLS connections
// created using its dialer and TLS dialer to always have a
// read watchdog timeout to address https://github.com/ooni/probe/issues/1609.
//
// QUIRK: the returned transport will always enforce 1 connection per host
// and we cannot get rid of this QUIRK requirement because it is
// necessary to perform sane measurements with tracing. We will be
// able to possibly relax this requirement after we change the
// way in which we perform measurements.
func NewHTTPTransportLegacy(logger model.DebugLogger, dialer model.Dialer, tlsDialer model.TLSDialer) model.HTTPTransport {
	return WrapHTTPTransport(logger, newOOHTTPBaseTransportLegacy(dialer, tlsDialer))
}

// newOOHTTPBaseTransportLegacy is the low-level factory used by NewHTTPTransport
// to create a new, suitable HTTPTransport for HTTP2 and HTTP/1.1.
//
// Deprecated: do not use this func in new code.
//
// This factory uses github.com/ooni/oohttp, hence its name.
func newOOHTTPBaseTransportLegacy(dialer model.Dialer, tlsDialer model.TLSDialer) model.HTTPTransport {
	// Using oohttp to support any TLS library.
	txp := oohttp.DefaultTransport.(*oohttp.Transport).Clone()

	// This wrapping ensures that we always have a timeout when we
	// are using HTTP; see https://github.com/ooni/probe/issues/1609.
	dialer = &httpDialerWithReadTimeout{dialer}
	txp.DialContext = dialer.DialContext
	tlsDialer = &httpTLSDialerWithReadTimeout{tlsDialer}
	txp.DialTLSContext = tlsDialer.DialTLSContext

	// We are using a different strategy to implement proxy: we
	// use a specific dialer that knows about proxying.
	txp.Proxy = nil

	// Better for Cloudflare DNS and also better because we have less
	// noisy events and we can better understand what happened.
	txp.MaxConnsPerHost = 1

	// The following (1) reduces the number of headers that Go will
	// automatically send for us and (2) ensures that we always receive
	// back the true headers, such as Content-Length. This change is
	// functional to OONI's goal of observing the network.
	txp.DisableCompression = true

	// Required to enable using HTTP/2 (which will be anyway forced
	// upon us when we are using TLS parroting).
	txp.ForceAttemptHTTP2 = true

	// Ensure we correctly forward CloseIdleConnections.
	return &httpTransportConnectionsCloser{
		HTTPTransport: &httpTransportStdlib{&oohttp.StdlibTransport{Transport: txp}},
		Dialer:        dialer,
		TLSDialer:     tlsDialer,
	}
}

// NewHTTPTransportStdlib creates a new HTTPTransport using
// the stdlib for DNS resolutions and TLS.
//
// Deprecated: do not use this func in new code.
//
// This factory calls NewHTTPTransportLegacy with suitable dialers.
func (netx *Netx) NewHTTPTransportStdlibLegacy(logger model.DebugLogger) model.HTTPTransport {
	dialer := netx.NewDialerWithResolver(logger, netx.NewStdlibResolver(logger))
	tlsDialer := NewTLSDialer(dialer, netx.NewTLSHandshakerStdlib(logger))
	return NewHTTPTransportLegacy(logger, dialer, tlsDialer)
}

// NewHTTPTransportStdlibLegacy is equivalent to creating an empty [*Netx]
// and calling its NewHTTPTransportStdlibLegacy method.
//
// Deprecated: do not use this func in new code.
func NewHTTPTransportStdlibLegacy(logger model.DebugLogger) model.HTTPTransport {
	netx := &Netx{Underlying: nil}
	return netx.NewHTTPTransportStdlibLegacy(logger)
}

// NewHTTPClientStdlibLegacy creates a new HTTPClient that uses the
// standard library for TLS and DNS resolutions.
//
// Deprecated: do not use this func in new code.
func NewHTTPClientStdlibLegacy(logger model.DebugLogger) model.HTTPClient {
	txp := NewHTTPTransportStdlibLegacy(logger)
	return NewHTTPClientLegacy(txp)
}

// NewHTTPClientWithResolverLegacy creates a new HTTPTransport using the
// given resolver and then from that builds an HTTPClient.
func NewHTTPClientWithResolverLegacy(logger model.Logger, reso model.Resolver) model.HTTPClient {
	return NewHTTPClientLegacy(NewHTTPTransportWithResolverLegacy(logger, reso))
}

// NewHTTPClientLegacy creates a new, wrapped HTTPClient using the given transport.
//
// The returned client would not have cookies support.
func NewHTTPClientLegacy(txp model.HTTPTransport) model.HTTPClient {
	return WrapHTTPClient(&http.Client{Transport: txp})
}
