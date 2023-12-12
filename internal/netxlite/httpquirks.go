package netxlite

//
// QUIRKy Legacy HTTP code and behavior assumed by ./legacy/netx 😅
//
// Ideally, we should not modify this code or apply minimal and obvious changes.
//
// TODO(https://github.com/ooni/probe/issues/2534)
//

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/feature/oohttpfeat"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// NewHTTPTransportWithResolver creates a new HTTP transport using
// the stdlib for everything but the given resolver.
//
// This function behavior is QUIRKY as documented in [NewHTTPTransport].
func NewHTTPTransportWithResolver(logger model.DebugLogger, reso model.Resolver) model.HTTPTransport {
	dialer := NewDialerWithResolver(logger, reso)
	thx := NewTLSHandshakerStdlib(logger)
	tlsDialer := NewTLSDialer(dialer, thx)
	return NewHTTPTransport(logger, dialer, tlsDialer)
}

// NewHTTPTransport returns a wrapped HTTP transport for HTTP2 and HTTP/1.1
// using the given dialer and logger.
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
func NewHTTPTransport(logger model.DebugLogger, dialer model.Dialer, tlsDialer model.TLSDialer) model.HTTPTransport {
	return WrapHTTPTransport(logger, newOOHTTPBaseTransport(dialer, tlsDialer))
}

// newOOHTTPBaseTransport is the low-level factory used by NewHTTPTransport
// to create a new, suitable HTTPTransport for HTTP2 and HTTP/1.1.
//
// This factory uses github.com/ooni/oohttp, hence its name.
//
// This function behavior is QUIRKY as documented in [NewHTTPTransport].
func newOOHTTPBaseTransport(dialer model.Dialer, tlsDialer model.TLSDialer) model.HTTPTransport {
	// Using oohttp to support any TLS library iff it's possible to do so, otherwise we
	// are going to use the standard library w/o using HTTP/2 support.
	txp := oohttpfeat.NewHTTPTransport()

	// This wrapping ensures that we always have a timeout when we
	// are using HTTP; see https://github.com/ooni/probe/issues/1609.
	dialer = &httpDialerWithReadTimeout{dialer}
	txp.SetDialContext(dialer.DialContext)
	tlsDialer = &httpTLSDialerWithReadTimeout{tlsDialer}
	txp.SetDialTLSContext(tlsDialer.DialTLSContext)

	// We are using a different strategy to implement proxy: we
	// use a specific dialer that knows about proxying.
	txp.SetProxy(nil)

	// Better for Cloudflare DNS and also better because we have less
	// noisy events and we can better understand what happened.
	txp.SetMaxConnsPerHost(1)

	// The following (1) reduces the number of headers that Go will
	// automatically send for us and (2) ensures that we always receive
	// back the true headers, such as Content-Length. This change is
	// functional to OONI's goal of observing the network.
	txp.SetDisableCompression(true)

	// We now rely on feature/oohttpfeat to decide whether it's
	// possible for us to actually enable HTTP/2.
	//
	//	txp.ForceAttemptHTTP2 = true
	//
	// Please, keep this comment until at least 3.21 because it provides
	// an historical documentation of how we changed the codebase.

	// Ensure we correctly forward CloseIdleConnections.
	return &httpTransportConnectionsCloser{
		HTTPTransport: &httpTransportStdlib{txp},
		Dialer:        dialer,
		TLSDialer:     tlsDialer,
	}
}

// NewHTTPTransportStdlib creates a new HTTPTransport using
// the stdlib for DNS resolutions and TLS.
//
// This factory calls NewHTTPTransport with suitable dialers.
//
// This function behavior is QUIRKY as documented in [NewHTTPTransport].
func (netx *Netx) NewHTTPTransportStdlib(logger model.DebugLogger) model.HTTPTransport {
	dialer := netx.NewDialerWithResolver(logger, netx.NewStdlibResolver(logger))
	tlsDialer := NewTLSDialer(dialer, netx.NewTLSHandshakerStdlib(logger))
	return NewHTTPTransport(logger, dialer, tlsDialer)
}

// NewHTTPTransportStdlib is equivalent to creating an empty [*Netx]
// and calling its NewHTTPTransportStdlib method.
//
// This function behavior is QUIRKY as documented in [NewHTTPTransport].
func NewHTTPTransportStdlib(logger model.DebugLogger) model.HTTPTransport {
	netx := &Netx{Underlying: nil}
	return netx.NewHTTPTransportStdlib(logger)
}

// NewHTTPClientStdlib creates a new HTTPClient that uses the
// standard library for TLS and DNS resolutions.
//
// This function behavior is QUIRKY as documented in [NewHTTPTransport].
func NewHTTPClientStdlib(logger model.DebugLogger) model.HTTPClient {
	txp := NewHTTPTransportStdlib(logger)
	return NewHTTPClient(txp)
}

// NewHTTPClientWithResolver creates a new HTTPTransport using the
// given resolver and then from that builds an HTTPClient.
//
// This function behavior is QUIRKY as documented in [NewHTTPTransport].
func NewHTTPClientWithResolver(logger model.Logger, reso model.Resolver) model.HTTPClient {
	return NewHTTPClient(NewHTTPTransportWithResolver(logger, reso))
}

// NewHTTPClient creates a new, wrapped HTTPClient using the given transport.
//
// This function behavior is QUIRKY because it does not configure a cookie jar, which
// is probably not the right thing to do in many cases, but legacy code MAY depend
// on this behavior. TODO(https://github.com/ooni/probe/issues/2534).
func NewHTTPClient(txp model.HTTPTransport) model.HTTPClient {
	return WrapHTTPClient(&http.Client{Transport: txp})
}
